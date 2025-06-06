package controller

import (
	"context"
	"fmt"
	"strings"

	n8nv1alpha1 "github.com/jakub-k-slys/n8n-operator/api/v1alpha1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const n8nVersion = "1.85.3"
const n8nFinalizer = "n8n.slys.dev/finalizer"
const n8nDockerImage = "ghcr.io/n8n-io/n8n:" + n8nVersion

const (
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

func (r *N8nReconciler) serviceForN8n(n8n *n8nv1alpha1.N8n) *corev1.Service {
	ls := labelsForN8n()
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      n8n.Name,
			Namespace: n8n.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{{
				Port:       80,
				TargetPort: intstr.FromString("http"),
				Protocol:   corev1.ProtocolTCP,
				Name:       "http",
			}},
			Selector: ls,
		},
	}
	ctrl.SetControllerReference(n8n, svc, r.Scheme)
	return svc
}

func (r *N8nReconciler) doFinalizerOperationsForN8n(cr *n8nv1alpha1.N8n) {
	r.Recorder.Event(cr, "Warning", "Deleting",
		fmt.Sprintf("Custom Resource %s is being deleted from the namespace %s",
			cr.Name,
			cr.Namespace))
}

func (r *N8nReconciler) deploymentForN8n(
	n8n *n8nv1alpha1.N8n) (*appsv1.Deployment, error) {
	ls := labelsForN8n()
	replicas := int32(1)
	image := n8nDockerImage
	var volumes []corev1.Volume
	var volumeMounts []corev1.VolumeMount

	if n8n.Spec.PersistentStorage != nil && n8n.Spec.PersistentStorage.Enable {
		volumes = append(volumes, corev1.Volume{
			Name: "n8n-data",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: n8n.Name + "-data",
				},
			},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "n8n-data",
			MountPath: "/home/node/.n8n",
		})

		if err := r.createPVCIfNotExists(n8n); err != nil {
			return nil, err
		}
	}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      n8n.Name,
			Namespace: n8n.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":    "n8n",
				"app":                       "n8n",
				"app.kubernetes.io/version": n8nVersion,
				"version":                   n8nVersion,
			},
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
					SecurityContext: getPodSecurityContext(),
					Volumes:         volumes,
					InitContainers: []corev1.Container{{
						Name:            "init-permissions",
						Image:           "busybox",
						ImagePullPolicy: corev1.PullIfNotPresent,
						Command: []string{
							"sh",
							"-c",
							"chown -R 1000:1000 /home/node/.n8n",
						},
						SecurityContext: &corev1.SecurityContext{
							RunAsUser:    &[]int64{0}[0], // Run as root to change ownership
							RunAsNonRoot: &[]bool{false}[0],
						},
						VolumeMounts: volumeMounts,
					}},
					Containers: []corev1.Container{{
						Image:           image,
						Name:            "n8n",
						ImagePullPolicy: corev1.PullIfNotPresent,
						SecurityContext: getContainerSecurityContext(),
						Ports: []corev1.ContainerPort{{
							ContainerPort: 5678,
							Name:          "http",
						}},
						Command:      []string{"tini", "--", "/docker-entrypoint.sh"},
						Env:          getN8nEnvVars(n8n),
						VolumeMounts: volumeMounts,
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

func labelsForN8n() map[string]string {
	var imageTag string
	image := n8nDockerImage
	imageTag = strings.Split(image, ":")[1]
	return map[string]string{"app.kubernetes.io/name": "n8n-operator",
		"app.kubernetes.io/version":    imageTag,
		"app.kubernetes.io/managed-by": "N8nController",
	}
}

func (r *N8nReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&n8nv1alpha1.N8n{}).
		Complete(r)
}
