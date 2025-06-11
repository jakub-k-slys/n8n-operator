package controller

import (
	n8nv1alpha1 "github.com/jakub-k-slys/n8n-operator/api/v1alpha1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *N8nReconciler) serviceMonitorForN8n(n8n *n8nv1alpha1.N8n) *monitoringv1.ServiceMonitor {
	labels := labelsForN8n()

	sm := &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      n8n.Name,
			Namespace: n8n.Namespace,
			Labels:    labels,
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Endpoints: []monitoringv1.Endpoint{
				{
					Port: "http",
					Path: "/metrics",
				},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: labels,
			},
		},
	}

	if err := ctrl.SetControllerReference(n8n, sm, r.Scheme); err != nil {
		return nil
	}
	return sm
}
