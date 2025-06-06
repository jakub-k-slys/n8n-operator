package controller

import (
	n8nv1alpha1 "github.com/jakub-k-slys/n8n-operator/api/v1alpha1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func (r *N8nReconciler) ingressForN8n(n8n *n8nv1alpha1.N8n) *networkingv1.Ingress {
	pathType := networkingv1.PathTypePrefix
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      n8n.Name,
			Namespace: n8n.Namespace,
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: &n8n.Spec.Ingress.IngressClassName,
			Rules: []networkingv1.IngressRule{
				{
					Host: n8n.Spec.Hostname.Url,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: n8n.Name,
											Port: networkingv1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if len(n8n.Spec.Ingress.TLS) > 0 {
		ing.Spec.TLS = make([]networkingv1.IngressTLS, len(n8n.Spec.Ingress.TLS))
		for i, tls := range n8n.Spec.Ingress.TLS {
			ing.Spec.TLS[i] = networkingv1.IngressTLS{
				Hosts:      tls.Hosts,
				SecretName: tls.SecretName,
			}
		}
	}

	ctrl.SetControllerReference(n8n, ing, r.Scheme)
	return ing
}

func (r *N8nReconciler) httpRouteForN8n(n8n *n8nv1alpha1.N8n) *gatewayv1.HTTPRoute {
	path := "/"
	serviceKind := gatewayv1.Kind("Service")
	portNumber := gatewayv1.PortNumber(80)
	var pathType gatewayv1.PathMatchType = "PathPrefix"
	route := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      n8n.Name,
			Namespace: n8n.Namespace,
		},
		Spec: gatewayv1.HTTPRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{
					{
						Name:      gatewayv1.ObjectName(n8n.Spec.HTTPRoute.GatewayRef.Name),
						Namespace: (*gatewayv1.Namespace)(&n8n.Spec.HTTPRoute.GatewayRef.Namespace),
					},
				},
			},
			Hostnames: []gatewayv1.Hostname{
				gatewayv1.Hostname(n8n.Spec.Hostname.Url),
			},
			Rules: []gatewayv1.HTTPRouteRule{
				{
					Matches: []gatewayv1.HTTPRouteMatch{
						{
							Path: &gatewayv1.HTTPPathMatch{
								Type:  &pathType,
								Value: &path,
							},
						},
					},
					BackendRefs: []gatewayv1.HTTPBackendRef{
						{
							BackendRef: gatewayv1.BackendRef{
								BackendObjectReference: gatewayv1.BackendObjectReference{
									Name: gatewayv1.ObjectName(n8n.Name),
									Port: &portNumber,
									Kind: &serviceKind,
								},
							},
						},
					},
				},
			},
		},
	}

	ctrl.SetControllerReference(n8n, route, r.Scheme)
	return route
}
