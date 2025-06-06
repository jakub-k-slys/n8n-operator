package controller

import (
	"context"
	"fmt"

	n8nv1alpha1 "github.com/jakub-k-slys/n8n-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	n8nFinalizer     = "n8n.slys.dev/finalizer"
	typeAvailableN8n = "Available"
	typeDegradedN8n  = "Degraded"
)

type N8nReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=n8n.slys.dev,resources=n8ns,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=n8n.slys.dev,resources=n8ns/status,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=httproutes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitors,verbs=get;list;watch;create;update;patch;delete

func (r *N8nReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	n8n := &n8nv1alpha1.N8n{}
	err := r.Get(ctx, req.NamespacedName, n8n)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("n8n resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get n8n")
		return ctrl.Result{}, err
	}

	// Initialize status conditions if not set
	if n8n.Status.Conditions == nil || len(n8n.Status.Conditions) == 0 {
		if err = r.updateStatus(ctx, n8n, typeAvailableN8n, metav1.ConditionUnknown, "Reconciling", "Starting reconciliation"); err != nil {
			log.Error(err, "Failed to update n8n status")
			return ctrl.Result{}, err
		}
		if err := r.Get(ctx, req.NamespacedName, n8n); err != nil {
			log.Error(err, "Failed to re-fetch n8n")
			return ctrl.Result{}, err
		}
	}

	// Handle finalizer
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

	// Handle deletion
	if n8n.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(n8n, n8nFinalizer) {
			if err = r.updateStatus(ctx, n8n, typeDegradedN8n, metav1.ConditionUnknown, "Finalizing",
				fmt.Sprintf("Performing finalizer operations for the custom resource: %s", n8n.Name)); err != nil {
				return ctrl.Result{}, err
			}

			r.doFinalizerOperationsForN8n(n8n)

			if err = r.updateStatus(ctx, n8n, typeDegradedN8n, metav1.ConditionTrue, "Finalizing",
				fmt.Sprintf("Finalizer operations for custom resource %s were successfully accomplished", n8n.Name)); err != nil {
				return ctrl.Result{}, err
			}

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

	// Reconcile Deployment
	if err := r.createOrUpdateDeployment(ctx, n8n); err != nil {
		return ctrl.Result{}, err
	}

	// Reconcile Service
	if err := r.createOrUpdateService(ctx, n8n); err != nil {
		return ctrl.Result{}, err
	}

	// Reconcile Ingress
	if err := r.createOrUpdateIngress(ctx, n8n); err != nil {
		return ctrl.Result{}, err
	}

	// Reconcile HTTPRoute
	if err := r.createOrUpdateHTTPRoute(ctx, n8n); err != nil {
		return ctrl.Result{}, err
	}

	// Reconcile ServiceMonitor
	if err := r.createOrUpdateServiceMonitor(ctx, n8n); err != nil {
		return ctrl.Result{}, err
	}

	// Update status
	if err := r.updateStatus(ctx, n8n, typeAvailableN8n, metav1.ConditionTrue, "Reconciling",
		fmt.Sprintf("Resources for custom resource (%s) reconciled successfully", n8n.Name)); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *N8nReconciler) doFinalizerOperationsForN8n(cr *n8nv1alpha1.N8n) {
	r.Recorder.Event(cr, "Warning", "Deleting",
		fmt.Sprintf("Custom Resource %s is being deleted from the namespace %s",
			cr.Name,
			cr.Namespace))
}

func (r *N8nReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&n8nv1alpha1.N8n{}).
		Complete(r)
}
