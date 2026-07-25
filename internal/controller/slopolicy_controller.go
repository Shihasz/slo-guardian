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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	srev1alpha1 "github.com/Shihasz/slo-guardian/api/v1alpha1"
)

// SLOPolicyReconciler reconciles a SLOPolicy object
type SLOPolicyReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	Tracker *Tracker
}

// +kubebuilder:rbac:groups=sre.sre.dev,resources=slopolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=sre.sre.dev,resources=slopolicies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=sre.sre.dev,resources=slopolicies/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the SLOPolicy object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.24.1/pkg/reconcile
func (r *SLOPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

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

	key := req.NamespacedName.String()
	availability, windowCount := r.Tracker.Record(key, success)

	log.Info("health check result",
		"name", policy.Name,
		"targetURL", policy.Spec.TargetURL,
		"success", success,
		"availabilityPercent", availability,
		"windowCount", windowCount,
	)

	interval := time.Duration(policy.Spec.CheckIntervalSeconds) * time.Second
	if interval <= 0 {
		interval = 10 * time.Second
	}
	return ctrl.Result{RequeueAfter: interval}, nil
}

// probeTarget does a simple HTTP GET with a short timeout and treats any
// 2xx/3xx response as healthy.
func probeTarget(url string) bool {
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode < 400
}

// SetupWithManager sets up the controller with the Manager.
func (r *SLOPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.Tracker == nil {
		r.Tracker = NewTracker()
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&srev1alpha1.SLOPolicy{}).
		Named("slopolicy").
		Complete(r)
}
