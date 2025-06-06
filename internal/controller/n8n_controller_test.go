/*
Copyright 2025.

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
	"time"

	cachev1alpha1 "github.com/jakub-k-slys/n8n-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("N8n Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		n8n := &cachev1alpha1.N8n{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind N8n")
			err := k8sClient.Get(ctx, typeNamespacedName, n8n)
			if err != nil && errors.IsNotFound(err) {
				resource := &cachev1alpha1.N8n{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: cachev1alpha1.N8nSpec{
						Hostname: &cachev1alpha1.HostnameConfig{
							Enable: true,
							Url:    "test.example.com",
						},
						Database: cachev1alpha1.Database{
							Postgres: cachev1alpha1.Postgres{
								Host:     "localhost",
								Port:     5432,
								Database: "n8n",
								User:     "n8n",
								Password: "n8n",
								Ssl:      false,
							},
						},
						PersistentStorage: &cachev1alpha1.PersistentStorageConfig{
							Enable:           true,
							Size:             "1Gi",
							StorageClassName: "standard",
						},
						Metrics: &cachev1alpha1.MetricsConfig{
							Enable: true,
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &cachev1alpha1.N8n{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance N8n")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &N8nReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Recorder: k8sManager.GetEventRecorderFor("n8n-controller"),
			}

			// Give the controller a chance to process the resource
			Eventually(func() error {
				_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: typeNamespacedName,
				})
				return err
			}, time.Second*5, time.Millisecond*100).Should(Succeed())

			// Wait a moment for resources to be created
			time.Sleep(time.Second)

			// Verify Deployment is created
			deployment := &appsv1.Deployment{}
			var err error
			err = k8sClient.Get(ctx, typeNamespacedName, deployment)
			Expect(err).NotTo(HaveOccurred())
			Expect(deployment.Spec.Template.Spec.Containers).To(HaveLen(1))
			Expect(deployment.Spec.Template.Spec.Containers[0].Image).To(Equal(n8nDockerImage))

			// Verify Service is created
			service := &corev1.Service{}
			err = k8sClient.Get(ctx, typeNamespacedName, service)
			Expect(err).NotTo(HaveOccurred())
			Expect(service.Spec.Ports).To(HaveLen(1))
			Expect(service.Spec.Ports[0].Port).To(Equal(int32(80)))

			// Verify PVC is created
			pvc := &corev1.PersistentVolumeClaim{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: resourceName + "-data", Namespace: "default"}, pvc)
			Expect(err).NotTo(HaveOccurred())
			Expect(pvc.Spec.Resources.Requests[corev1.ResourceStorage]).To(Equal(resource.MustParse("1Gi")))

			// Verify ServiceMonitor is created
			sm := &monitoringv1.ServiceMonitor{}
			err = k8sClient.Get(ctx, typeNamespacedName, sm)
			Expect(err).NotTo(HaveOccurred())
			Expect(sm.Spec.Endpoints).To(HaveLen(1))
			Expect(sm.Spec.Endpoints[0].Path).To(Equal("/metrics"))
		})
	})
})
