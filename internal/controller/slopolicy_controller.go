/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"net/http"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	srev1alpha1 "github.com/Shihasz/slo-guardian/api/v1alpha1"
)

// SLOPolicyReconciler reconciles a SLOPolicy object
type SLOPolicyReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Tracker  *Tracker
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=sre.sre.dev,resources=slopolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=sre.sre.dev,resources=slopolicies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=sre.sre.dev,resources=slopolicies/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *SLOPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	if r.Tracker == nil {
		r.Tracker = NewTracker()
	}

	var policy srev1alpha1.SLOPolicy
	if err := r.Get(ctx, req.NamespacedName, &policy); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("SLOPolicy deleted, nothing to do", "name", req.Name)
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch SLOPolicy")
		return ctrl.Result{}, err
	}

	success := probeTarget(policy.Spec.TargetURL)

	key := req.String()
	availability, windowCount := r.Tracker.Record(key, success)

	policy.Status.TotalChecks++
	if !success {
		policy.Status.FailedChecks++
	}
	policy.Status.CurrentAvailabilityPercent = availability
	policy.Status.ErrorBudgetRemainingPercent = computeErrorBudgetRemaining(availability, policy.Spec.SLOTargetPercent)
	now := metav1.Now()
	policy.Status.LastCheckTime = now

	availabilityGauge.WithLabelValues(policy.Name, policy.Namespace).Set(availability)
	errorBudgetGauge.WithLabelValues(policy.Name, policy.Namespace).Set(policy.Status.ErrorBudgetRemainingPercent)

	resultLabel := "success"
	if !success {
		resultLabel = "failure"
	}
	checksTotal.WithLabelValues(policy.Name, policy.Namespace, resultLabel).Inc()

	if policy.Status.ErrorBudgetRemainingPercent < 0 && windowCount >= 1 {
		if r.canRemediate(&policy, now) {
			if err := r.remediate(ctx, &policy); err != nil {
				log.Error(err, "remediation failed")
			} else {
				policy.Status.LastRemediationTime = &now
				remediationsTotal.WithLabelValues(policy.Name, policy.Namespace, policy.Spec.RemediationAction).Inc()
			}
		}
	}

	if err := r.Status().Update(ctx, &policy); err != nil {
		log.Error(err, "unable to update SLOPolicy status")
		return ctrl.Result{}, err
	}

	log.Info("health check result",
		"name", policy.Name,
		"success", success,
		"availabilityPercent", availability,
		"errorBudgetRemainingPercent", policy.Status.ErrorBudgetRemainingPercent,
		"windowCount", windowCount,
	)

	interval := time.Duration(policy.Spec.CheckIntervalSeconds) * time.Second
	if interval <= 0 {
		interval = 10 * time.Second
	}
	return ctrl.Result{RequeueAfter: interval}, nil
}

// computeErrorBudgetRemaining returns the percentage of the allowed failure
// budget that remains. 100 = full budget untouched, 0 = fully consumed,
// negative = SLO breached (over budget).
func computeErrorBudgetRemaining(availability, sloTarget float64) float64 {
	allowedFailure := 100.0 - sloTarget
	if allowedFailure <= 0 {
		return 0
	}
	actualFailure := 100.0 - availability
	remaining := allowedFailure - actualFailure
	return (remaining / allowedFailure) * 100.0
}

// probeTarget does a simple HTTP GET with a short timeout and treats any
// 2xx/3xx response as healthy.
func probeTarget(url string) bool {
	httpClient := http.Client{Timeout: 3 * time.Second}
	resp, err := httpClient.Get(url)
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode < 400
}

// canRemediate checks whether enough time has passed since the last
// remediation attempt to avoid repeatedly restarting/scaling the target.
func (r *SLOPolicyReconciler) canRemediate(policy *srev1alpha1.SLOPolicy, now metav1.Time) bool {
	if policy.Spec.RemediationAction == "None" || policy.Spec.RemediationAction == "" {
		return false
	}
	if policy.Status.LastRemediationTime == nil {
		return true
	}
	cooldown := time.Duration(policy.Spec.RemediationCooldownSeconds) * time.Second
	if cooldown <= 0 {
		cooldown = 60 * time.Second
	}
	return now.Sub(policy.Status.LastRemediationTime.Time) >= cooldown
}

// remediate applies the configured remediation action against the target Deployment.
func (r *SLOPolicyReconciler) remediate(ctx context.Context, policy *srev1alpha1.SLOPolicy) error {
	var deploy appsv1.Deployment
	deployKey := types.NamespacedName{Name: policy.Spec.TargetDeployment, Namespace: policy.Namespace}
	if err := r.Get(ctx, deployKey, &deploy); err != nil {
		return err
	}

	switch policy.Spec.RemediationAction {
	case "RestartDeployment":
		if deploy.Spec.Template.Annotations == nil {
			deploy.Spec.Template.Annotations = map[string]string{}
		}
		deploy.Spec.Template.Annotations["slo-guardian/restartedAt"] = time.Now().Format(time.RFC3339)
		if err := r.Update(ctx, &deploy); err != nil {
			return err
		}
		r.Recorder.Eventf(policy, corev1.EventTypeWarning, "Remediated",
			"Restarted deployment %s after error budget breach", deploy.Name)

	case "ScaleUp":
		var replicas int32 = 1
		if deploy.Spec.Replicas != nil {
			replicas = *deploy.Spec.Replicas
		}
		replicas++
		deploy.Spec.Replicas = &replicas
		if err := r.Update(ctx, &deploy); err != nil {
			return err
		}
		r.Recorder.Eventf(policy, corev1.EventTypeWarning, "Remediated",
			"Scaled deployment %s to %d replicas after error budget breach", deploy.Name, replicas)
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SLOPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.Tracker == nil {
		r.Tracker = NewTracker()
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&srev1alpha1.SLOPolicy{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Named("slopolicy").
		Complete(r)
}
