package controller

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cachev1alpha1 "github.com/jakub-k-slys/n8n-operator/api/v1alpha1"
)

const n8nFinalizer = "cache.slys.dev/finalizer"

const (
	typeAvailableN8n = "Available"
	typeDegradedN8n  = "Degraded"
)

type N8nReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=cache.slys.dev,resources=n8ns,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cache.slys.dev,resources=n8ns/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cache.slys.dev,resources=n8ns/finalizers,verbs=update
func (r *N8nReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	n8n := &cachev1alpha1.N8n{}
	err := r.Get(ctx, req.NamespacedName, n8n)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("n8n resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get n8n")
		return ctrl.Result{}, err
	}

	if n8n.Status.Conditions == nil || len(n8n.Status.Conditions) == 0 {
		meta.SetStatusCondition(&n8n.Status.Conditions, metav1.Condition{Type: typeAvailableN8n, Status: metav1.ConditionUnknown, Reason: "Reconciling", Message: "Starting reconciliation"})
		if err = r.Status().Update(ctx, n8n); err != nil {
			log.Error(err, "Failed to update n8n status")
			return ctrl.Result{}, err
		}
		if err := r.Get(ctx, req.NamespacedName, n8n); err != nil {
			log.Error(err, "Failed to re-fetch n8n")
			return ctrl.Result{}, err
		}
	}

	if !controllerutil.ContainsFinalizer(n8n, n8nFinalizer) {
		log.Info("Adding Finalizer for n8n")
		if ok := controllerutil.AddFinalizer(n8n, n8nFinalizer); !ok {
			log.Error(err, "Failed to add finalizer into the custom resource")
			return ctrl.Result{Requeue: true}, nil
		}

		if err = r.Update(ctx, n8n); err != nil {
			log.Error(err, "Failed to update custom resource to add finalizer")
			return ctrl.Result{}, err
		}
	}
	isN8nMarkedToBeDeleted := n8n.GetDeletionTimestamp() != nil
	if isN8nMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(n8n, n8nFinalizer) {
			log.Info("Performing Finalizer Operations for n8n before delete CR")
			meta.SetStatusCondition(&n8n.Status.Conditions, metav1.Condition{Type: typeDegradedN8n,
				Status: metav1.ConditionUnknown, Reason: "Finalizing",
				Message: fmt.Sprintf("Performing finalizer operations for the custom resource: %s ", n8n.Name)})
			if err := r.Status().Update(ctx, n8n); err != nil {
				log.Error(err, "Failed to update N8n status")
				return ctrl.Result{}, err
			}

			r.doFinalizerOperationsForN8n(n8n)
			if err := r.Get(ctx, req.NamespacedName, n8n); err != nil {
				log.Error(err, "Failed to re-fetch N8n")
				return ctrl.Result{}, err
			}

			meta.SetStatusCondition(&n8n.Status.Conditions, metav1.Condition{Type: typeDegradedN8n,
				Status: metav1.ConditionTrue, Reason: "Finalizing",
				Message: fmt.Sprintf("Finalizer operations for custom resource %s name were successfully accomplished", n8n.Name)})
			if err := r.Status().Update(ctx, n8n); err != nil {
				log.Error(err, "Failed to update N8n status")
				return ctrl.Result{}, err
			}
			log.Info("Removing Finalizer for n8n after successfully perform the operations")

			if ok := controllerutil.RemoveFinalizer(n8n, n8nFinalizer); !ok {
				log.Error(err, "Failed to remove finalizer for n8n")
				return ctrl.Result{Requeue: true}, nil
			}

			if err := r.Update(ctx, n8n); err != nil {
				log.Error(err, "Failed to remove finalizer for n8n")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	found := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: n8n.Name, Namespace: n8n.Namespace}, found)

	if err != nil && apierrors.IsNotFound(err) {
		dep, err := r.deploymentForN8n(n8n)
		if err != nil {
			log.Error(err, "Failed to define new Deployment resource for n8n")
			meta.SetStatusCondition(&n8n.Status.Conditions, metav1.Condition{Type: typeAvailableN8n,
				Status: metav1.ConditionFalse, Reason: "Reconciling",
				Message: fmt.Sprintf("Failed to create Deployment for the custom resource (%s): (%s)", n8n.Name, err)})
			if err := r.Status().Update(ctx, n8n); err != nil {
				log.Error(err, "Failed to update n8n status")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, err
		}
		log.Info("Creating a new Deployment",
			"Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)

		if err = r.Create(ctx, dep); err != nil {
			log.Error(err, "Failed to create new Deployment",
				"Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return ctrl.Result{}, err
		}

		return ctrl.Result{RequeueAfter: time.Minute}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Deployment")
		return ctrl.Result{}, err
	}

	size := n8n.Spec.Size
	if *found.Spec.Replicas != size {
		found.Spec.Replicas = &size
		if err = r.Update(ctx, found); err != nil {
			log.Error(err, "Failed to update Deployment",
				"Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)

			if err := r.Get(ctx, req.NamespacedName, n8n); err != nil {
				log.Error(err, "Failed to re-fetch n8n")
				return ctrl.Result{}, err
			}

			meta.SetStatusCondition(&n8n.Status.Conditions, metav1.Condition{Type: typeAvailableN8n,
				Status: metav1.ConditionFalse, Reason: "Resizing",
				Message: fmt.Sprintf("Failed to update the size for the custom resource (%s): (%s)", n8n.Name, err)})

			if err := r.Status().Update(ctx, n8n); err != nil {
				log.Error(err, "Failed to update N8n status")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}
	meta.SetStatusCondition(&n8n.Status.Conditions, metav1.Condition{Type: typeAvailableN8n,
		Status: metav1.ConditionTrue, Reason: "Reconciling",
		Message: fmt.Sprintf("Deployment for custom resource (%s) with %d replicas created successfully", n8n.Name, size)})

	if err := r.Status().Update(ctx, n8n); err != nil {
		log.Error(err, "Failed to update N8n status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *N8nReconciler) doFinalizerOperationsForN8n(cr *cachev1alpha1.N8n) {
	r.Recorder.Event(cr, "Warning", "Deleting",
		fmt.Sprintf("Custom Resource %s is being deleted from the namespace %s",
			cr.Name,
			cr.Namespace))
}

func (r *N8nReconciler) deploymentForN8n(
	n8n *cachev1alpha1.N8n) (*appsv1.Deployment, error) {
	ls := labelsForN8n(n8n.Name)
	replicas := n8n.Spec.Size
	image, err := imageForN8n()
	if err != nil {
		return nil, err
	}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      n8n.Name,
			Namespace: n8n.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: &[]bool{true}[0],
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
					Containers: []corev1.Container{{
						Image:           image,
						Name:            "n8n",
						ImagePullPolicy: corev1.PullIfNotPresent,
						SecurityContext: &corev1.SecurityContext{
							RunAsNonRoot:             &[]bool{true}[0],
							RunAsUser:                &[]int64{1001}[0],
							AllowPrivilegeEscalation: &[]bool{false}[0],
							Capabilities: &corev1.Capabilities{
								Drop: []corev1.Capability{
									"ALL",
								},
							},
						},
						Ports: []corev1.ContainerPort{{
							ContainerPort: 8080,
							Name:          "n8n",
						}},
						Command: []string{"n8n", "-m=64", "-o", "modern", "-v"},
					}},
				},
			},
		},
	}
	if err := ctrl.SetControllerReference(n8n, dep, r.Scheme); err != nil {
		return nil, err
	}
	return dep, nil
}

func labelsForN8n(name string) map[string]string {
	var imageTag string
	image, err := imageForN8n()
	if err == nil {
		imageTag = strings.Split(image, ":")[1]
	}
	return map[string]string{"app.kubernetes.io/name": "n8n-operator",
		"app.kubernetes.io/version":    imageTag,
		"app.kubernetes.io/managed-by": "N8nController",
	}
}

func imageForN8n() (string, error) {
	var imageEnvVar = "N8N_IMAGE"
	image, found := os.LookupEnv(imageEnvVar)
	if !found {
		return "", fmt.Errorf("Unable to find %s environment variable with the image", imageEnvVar)
	}
	return image, nil
}

func (r *N8nReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cachev1alpha1.N8n{}).
		Complete(r)
}
