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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

var _ = Describe("N8n Controller", func() {
	const resourceName = "test-resource"
	var (
		ctx                context.Context
		typeNamespacedName types.NamespacedName
		reconciler         *N8nReconciler
	)

	BeforeEach(func() {
		ctx = context.Background()
		typeNamespacedName = types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		reconciler = &N8nReconciler{
			Client:   k8sClient,
			Scheme:   k8sClient.Scheme(),
			Recorder: k8sManager.GetEventRecorderFor("n8n-controller"),
		}

		// Clean up any existing resources
		existing := &cachev1alpha1.N8n{}
		_ = k8sClient.Get(ctx, typeNamespacedName, existing)
		_ = k8sClient.Delete(ctx, existing)

		// Wait for deletion to complete
		Eventually(func() bool {
			err := k8sClient.Get(ctx, typeNamespacedName, &cachev1alpha1.N8n{})
			return errors.IsNotFound(err)
		}, time.Second*10, time.Millisecond*100).Should(BeTrue())

		// Wait a moment to ensure all resources are cleaned up
		time.Sleep(time.Second * 2)
	})

	AfterEach(func() {
		// Clean up any remaining resources
		existing := &cachev1alpha1.N8n{}
		_ = k8sClient.Get(ctx, typeNamespacedName, existing)
		_ = k8sClient.Delete(ctx, existing)

		// Wait for deletion to complete
		Eventually(func() bool {
			err := k8sClient.Get(ctx, typeNamespacedName, &cachev1alpha1.N8n{})
			return errors.IsNotFound(err)
		}, time.Second*10, time.Millisecond*100).Should(BeTrue())

		// Wait a moment to ensure all resources are cleaned up
		time.Sleep(time.Second * 2)
	})
	Context("When reconciling a resource with all features enabled", func() {
		It("should successfully reconcile the resource", func() {
			By("creating the custom resource with all features enabled")
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
			By("Reconciling the created resource")
			controllerReconciler := &N8nReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Recorder: k8sManager.GetEventRecorderFor("n8n-controller"),
			}

			// Wait for reconciliation to complete
			Eventually(func() error {
				_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: typeNamespacedName,
				})
				return err
			}, time.Second*10, time.Millisecond*100).Should(Succeed())

			// Verify Deployment is created
			Eventually(func() bool {
				deployment := &appsv1.Deployment{}
				if err := k8sClient.Get(ctx, typeNamespacedName, deployment); err != nil {
					return false
				}
				return len(deployment.Spec.Template.Spec.Containers) == 1 &&
					deployment.Spec.Template.Spec.Containers[0].Image == n8nDockerImage
			}, time.Second*5, time.Millisecond*100).Should(BeTrue())

			// Verify Service is created
			Eventually(func() bool {
				service := &corev1.Service{}
				if err := k8sClient.Get(ctx, typeNamespacedName, service); err != nil {
					return false
				}
				return len(service.Spec.Ports) == 1 &&
					service.Spec.Ports[0].Port == int32(80)
			}, time.Second*5, time.Millisecond*100).Should(BeTrue())

			// Verify PVC is created
			Eventually(func() bool {
				pvc := &corev1.PersistentVolumeClaim{}
				if err := k8sClient.Get(ctx, types.NamespacedName{Name: resourceName + "-data", Namespace: "default"}, pvc); err != nil {
					return false
				}
				quantity := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
				return quantity.String() == "1Gi"
			}, time.Second*5, time.Millisecond*100).Should(BeTrue())

			// Verify ServiceMonitor is created
			Eventually(func() bool {
				sm := &monitoringv1.ServiceMonitor{}
				if err := k8sClient.Get(ctx, typeNamespacedName, sm); err != nil {
					return false
				}
				return len(sm.Spec.Endpoints) == 1 &&
					sm.Spec.Endpoints[0].Path == "/metrics"
			}, time.Second*5, time.Millisecond*100).Should(BeTrue())
		})
	})

	Context("When reconciling a resource without metrics", func() {
		It("should not create ServiceMonitor", func() {
			By("creating the custom resource without metrics")
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
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			// Wait for reconciliation to complete and verify ServiceMonitor is not created
			Eventually(func() bool {
				// Reconcile
				_, err := reconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: typeNamespacedName,
				})
				if err != nil {
					return false
				}

				// Check ServiceMonitor
				sm := &monitoringv1.ServiceMonitor{}
				err = k8sClient.Get(ctx, typeNamespacedName, sm)
				return errors.IsNotFound(err)
			}, time.Second*15, time.Millisecond*100).Should(BeTrue(), "ServiceMonitor should not exist")

			// Cleanup
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
	})

	Context("When reconciling a resource with HTTPRoute", func() {
		It("should create HTTPRoute", func() {
			By("creating the custom resource with HTTPRoute")
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
					HTTPRoute: &cachev1alpha1.HTTPRouteConfig{
						Enable: true,
						GatewayRef: cachev1alpha1.GatewayRef{
							Name:      "test-gateway",
							Namespace: "default",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			// Wait for initial reconciliation
			Eventually(func() error {
				_, err := reconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: typeNamespacedName,
				})
				return err
			}, time.Second*10, time.Millisecond*100).Should(Succeed())

			// Verify HTTPRoute is created
			Eventually(func() bool {
				route := &gatewayv1.HTTPRoute{}
				if err := k8sClient.Get(ctx, typeNamespacedName, route); err != nil {
					return false
				}
				return len(route.Spec.Hostnames) > 0 && route.Spec.Hostnames[0] == gatewayv1.Hostname("test.example.com")
			}, time.Second*5, time.Millisecond*100).Should(BeTrue())

			// Cleanup
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
	})

	Context("When updating a resource", func() {
		BeforeEach(func() {
			By("creating the custom resource")
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
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())
		})

		It("should handle spec updates", func() {
			// Wait for initial reconciliation
			Eventually(func() error {
				_, err := reconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: typeNamespacedName,
				})
				return err
			}, time.Second*10, time.Millisecond*100).Should(Succeed())

			// Update the resource with metrics enabled
			Eventually(func() error {
				updated := &cachev1alpha1.N8n{}
				if err := k8sClient.Get(ctx, typeNamespacedName, updated); err != nil {
					return err
				}
				updated.Spec.Metrics = &cachev1alpha1.MetricsConfig{
					Enable: true,
				}
				return k8sClient.Update(ctx, updated)
			}, time.Second*5, time.Millisecond*100).Should(Succeed())

			// Wait for reconciliation after update
			Eventually(func() error {
				_, err := reconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: typeNamespacedName,
				})
				return err
			}, time.Second*10, time.Millisecond*100).Should(Succeed())

			// Verify ServiceMonitor is created
			Eventually(func() bool {
				sm := &monitoringv1.ServiceMonitor{}
				if err := k8sClient.Get(ctx, typeNamespacedName, sm); err != nil {
					return false
				}
				return len(sm.Spec.Endpoints) == 1 &&
					sm.Spec.Endpoints[0].Path == "/metrics"
			}, time.Second*5, time.Millisecond*100).Should(BeTrue())
		})
	})

	Context("When resource creation fails", func() {
		It("should validate required fields", func() {
			By("creating the custom resource with invalid configuration")
			resource := &cachev1alpha1.N8n{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: cachev1alpha1.N8nSpec{
					Database: cachev1alpha1.Database{
						Postgres: cachev1alpha1.Postgres{
							Host:     "", // Invalid empty host
							Port:     0,  // Invalid port
							Database: "", // Invalid empty database
							User:     "", // Invalid empty user
							Password: "", // Invalid empty password
						},
					},
				},
			}

			err := k8sClient.Create(ctx, resource)
			Expect(err).To(HaveOccurred())
			statusErr, ok := err.(*errors.StatusError)
			Expect(ok).To(BeTrue())
			Expect(statusErr.Status().Code).To(Equal(int32(422)))
			Expect(statusErr.Status().Reason).To(Equal(metav1.StatusReasonInvalid))
			Expect(len(statusErr.Status().Details.Causes)).To(BeNumerically(">", 0))
		})
	})

	Context("When deleting a resource", func() {
		It("should handle deletion", func() {
			By("creating the custom resource")
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
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			// Wait for initial reconciliation
			Eventually(func() error {
				_, err := reconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: typeNamespacedName,
				})
				return err
			}, time.Second*10, time.Millisecond*100).Should(Succeed())

			// Delete the resource
			Eventually(func() error {
				n8n := &cachev1alpha1.N8n{}
				if err := k8sClient.Get(ctx, typeNamespacedName, n8n); err != nil {
					return err
				}
				return k8sClient.Delete(ctx, n8n)
			}, time.Second*5, time.Millisecond*100).Should(Succeed())

			// Verify deletion handling
			Eventually(func() bool {
				err := k8sClient.Get(ctx, typeNamespacedName, &cachev1alpha1.N8n{})
				return errors.IsNotFound(err)
			}, time.Second*10, time.Millisecond*100).Should(BeTrue())
		})
	})
})
