package controller

import (
	"context"
	"fmt"

	n8nv1alpha1 "github.com/jakub-k-slys/n8n-operator/api/v1alpha1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// reconcileResource handles the common pattern of getting/creating a resource
func (r *N8nReconciler) reconcileResource(ctx context.Context, n8n *n8nv1alpha1.N8n, resource client.Object, createFn func() error) error {
	err := r.Get(ctx, types.NamespacedName{Name: n8n.Name, Namespace: n8n.Namespace}, resource)
	if err != nil && apierrors.IsNotFound(err) {
		if err := createFn(); err != nil {
			return fmt.Errorf("failed to create resource: %w", err)
		}
		return nil
	}
	return err
}

// updateStatus handles updating the status conditions of the N8n resource
func (r *N8nReconciler) updateStatus(ctx context.Context, n8n *n8nv1alpha1.N8n, conditionType string, status metav1.ConditionStatus, reason, message string) error {
	meta.SetStatusCondition(&n8n.Status.Conditions, metav1.Condition{
		Type:    conditionType,
		Status:  status,
		Reason:  reason,
		Message: message,
	})
	return r.Status().Update(ctx, n8n)
}

// handleResourceError updates the status condition when an error occurs
func (r *N8nReconciler) handleResourceError(ctx context.Context, n8n *n8nv1alpha1.N8n, err error, resourceType string) error {
	if err := r.updateStatus(ctx, n8n, typeAvailableN8n,
		metav1.ConditionFalse,
		"Reconciling",
		fmt.Sprintf("Failed to manage %s for the custom resource (%s): %v", resourceType, n8n.Name, err)); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}
	return err
}

// createOrUpdateDeployment handles the deployment reconciliation
func (r *N8nReconciler) createOrUpdateDeployment(ctx context.Context, n8n *n8nv1alpha1.N8n) error {
	return r.reconcileResource(ctx, n8n, &appsv1.Deployment{}, func() error {
		dep, err := r.deploymentForN8n(n8n)
		if err != nil {
			return r.handleResourceError(ctx, n8n, err, "Deployment")
		}
		return r.Create(ctx, dep)
	})
}

// createOrUpdateService handles the service reconciliation
func (r *N8nReconciler) createOrUpdateService(ctx context.Context, n8n *n8nv1alpha1.N8n) error {
	return r.reconcileResource(ctx, n8n, &corev1.Service{}, func() error {
		svc := r.serviceForN8n(n8n)
		return r.Create(ctx, svc)
	})
}

// createOrUpdateIngress handles the ingress reconciliation
func (r *N8nReconciler) createOrUpdateIngress(ctx context.Context, n8n *n8nv1alpha1.N8n) error {
	if n8n.Spec.Ingress == nil || !n8n.Spec.Ingress.Enable {
		return nil
	}
	return r.reconcileResource(ctx, n8n, &networkingv1.Ingress{}, func() error {
		ing := r.ingressForN8n(n8n)
		return r.Create(ctx, ing)
	})
}

// createOrUpdateHTTPRoute handles the HTTPRoute reconciliation
func (r *N8nReconciler) createOrUpdateHTTPRoute(ctx context.Context, n8n *n8nv1alpha1.N8n) error {
	if n8n.Spec.HTTPRoute == nil || !n8n.Spec.HTTPRoute.Enable {
		return nil
	}
	return r.reconcileResource(ctx, n8n, &gatewayv1.HTTPRoute{}, func() error {
		route := r.httpRouteForN8n(n8n)
		return r.Create(ctx, route)
	})
}

// createOrUpdateServiceMonitor handles the ServiceMonitor reconciliation
// createPVCIfNotExists creates a PVC for n8n data if it doesn't exist
func (r *N8nReconciler) createPVCIfNotExists(n8n *n8nv1alpha1.N8n) error {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      n8n.Name + "-data",
			Namespace: n8n.Namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(n8n.Spec.PersistentStorage.Size),
				},
			},
		},
	}

	if n8n.Spec.PersistentStorage.StorageClassName != "" {
		pvc.Spec.StorageClassName = &n8n.Spec.PersistentStorage.StorageClassName
	}

	if err := ctrl.SetControllerReference(n8n, pvc, r.Scheme); err != nil {
		return err
	}

	existingPvc := &corev1.PersistentVolumeClaim{}
	err := r.Get(context.TODO(), types.NamespacedName{Name: pvc.Name, Namespace: pvc.Namespace}, existingPvc)
	if err != nil && apierrors.IsNotFound(err) {
		if err := r.Create(context.TODO(), pvc); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	return nil
}

func (r *N8nReconciler) createOrUpdateServiceMonitor(ctx context.Context, n8n *n8nv1alpha1.N8n) error {
	if n8n.Spec.Metrics == nil || !n8n.Spec.Metrics.Enable {
		return nil
	}
	return r.reconcileResource(ctx, n8n, &monitoringv1.ServiceMonitor{}, func() error {
		sm := r.serviceMonitorForN8n(n8n)
		return r.Create(ctx, sm)
	})
}
