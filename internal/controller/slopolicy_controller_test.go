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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	srev1alpha1 "github.com/Shihasz/slo-guardian/api/v1alpha1"
)

var _ = Describe("SLOPolicy Controller", func() {
	Context("When reconciling a resource", func() {
		const (
			resourceName      = "test-resource"
			resourceNamespace = "default"
		)

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: resourceNamespace,
		}
		slopolicy := &srev1alpha1.SLOPolicy{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind SLOPolicy")
			err := k8sClient.Get(ctx, typeNamespacedName, slopolicy)
			if err != nil && errors.IsNotFound(err) {
				resource := &srev1alpha1.SLOPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: srev1alpha1.SLOPolicySpec{
						TargetDeployment:     "nginx-demo",
						TargetURL:            "http://localhost:8080",
						SLOTargetPercent:     99.9,
						CheckIntervalSeconds: 10,
						RemediationAction:    "None",
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &srev1alpha1.SLOPolicy{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance SLOPolicy")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &SLOPolicyReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Tracker:  NewTracker(),
				Recorder: record.NewFakeRecorder(10),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking the SLOPolicy status was updated after reconcile")
			updated := &srev1alpha1.SLOPolicy{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, typeNamespacedName, updated)
				return err == nil && updated.Status.TotalChecks > 0
			}, "5s", "200ms").Should(BeTrue())

			Expect(updated.Status.TotalChecks).To(Equal(int64(1)))
			Expect(updated.Status.FailedChecks).To(Equal(int64(1)))
			Expect(updated.Status.CurrentAvailabilityPercent).To(Equal(0.0))
		})
	})
})
