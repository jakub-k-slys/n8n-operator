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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cachev1alpha1 "github.com/jakub-k-slys/n8n-operator/api/v1alpha1"
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
					// TODO(user): Specify other spec details if needed.
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
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})
	})

	Context("When testing environment variable construction", func() {
		It("should set only N8N_USER_FOLDER when no database is configured", func() {
			n8n := &cachev1alpha1.N8n{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-n8n",
					Namespace: "default",
				},
				Spec: cachev1alpha1.N8nSpec{
					// No database configuration
				},
			}

			envVars := BuildEnvVars(n8n)

			Expect(envVars).To(HaveLen(1))
			Expect(envVars[0].Name).To(Equal("N8N_USER_FOLDER"))
			Expect(envVars[0].Value).To(Equal("/home/node"))
		})

		It("should set database environment variables when database is configured", func() {
			n8n := &cachev1alpha1.N8n{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-n8n",
					Namespace: "default",
				},
				Spec: cachev1alpha1.N8nSpec{
					Database: cachev1alpha1.Database{
						Postgres: cachev1alpha1.Postgres{
							Host:     "postgres-host",
							Port:     5432,
							Database: "n8n",
							User:     "n8n-user",
							Password: "password",
							Ssl:      true,
						},
					},
				},
			}

			envVars := BuildEnvVars(n8n)

			Expect(envVars).To(HaveLen(8))

			// Check that all expected environment variables are present
			envMap := make(map[string]string)
			for _, env := range envVars {
				envMap[env.Name] = env.Value
			}

			Expect(envMap["N8N_USER_FOLDER"]).To(Equal("/home/node"))
			Expect(envMap["DB_TYPE"]).To(Equal("postgresdb"))
			Expect(envMap["DB_POSTGRESDB_HOST"]).To(Equal("postgres-host"))
			Expect(envMap["DB_POSTGRESDB_PORT"]).To(Equal("5432"))
			Expect(envMap["DB_POSTGRESDB_DATABASE"]).To(Equal("n8n"))
			Expect(envMap["DB_POSTGRESDB_USER"]).To(Equal("n8n-user"))
			Expect(envMap["DB_POSTGRESDB_PASSWORD"]).To(Equal("password"))
			Expect(envMap["DB_POSTGRESDB_SSL_REJECT_UNAUTHORIZED"]).To(Equal("false")) // SSL is true, so reject unauthorized should be false
		})
	})
})
