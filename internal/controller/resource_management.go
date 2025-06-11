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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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

func (r *N8nReconciler) createOrUpdateServiceMonitor(ctx context.Context, n8n *n8nv1alpha1.N8n) error {
	sm := &monitoringv1.ServiceMonitor{}
	err := r.Get(ctx, types.NamespacedName{Name: n8n.Name, Namespace: n8n.Namespace}, sm)

	// If metrics are disabled, delete the ServiceMonitor if it exists
	if n8n.Spec.Metrics == nil || !n8n.Spec.Metrics.Enable {
		if err == nil {
			return r.Delete(ctx, sm)
		}
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	// Create ServiceMonitor if it doesn't exist
	if apierrors.IsNotFound(err) {
		sm = r.serviceMonitorForN8n(n8n)
		return r.Create(ctx, sm)
	}
	return err
}
