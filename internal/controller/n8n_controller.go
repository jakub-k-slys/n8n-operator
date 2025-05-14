package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	n8nv1alpha1 "github.com/jakub-k-slys/n8n-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const n8nFinalizer = "n8n.slys.dev/finalizer"
const n8nDockerImage = "ghcr.io/n8n-io/n8n:1.85.3"

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

	defaultReplicas := int32(1)
	if *found.Spec.Replicas != defaultReplicas {
		found.Spec.Replicas = &defaultReplicas
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
		Message: fmt.Sprintf("Deployment for custom resource (%s) created successfully", n8n.Name)})

	if err := r.Status().Update(ctx, n8n); err != nil {
		log.Error(err, "Failed to update N8n status")
		return ctrl.Result{}, err
	}

	// Check if the service already exists, if not create a new one
	service := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: n8n.Name, Namespace: n8n.Namespace}, service)
	if err != nil && apierrors.IsNotFound(err) {
		svc := r.serviceForN8n(n8n)
		log.Info("Creating a new Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
		err = r.Create(ctx, svc)
		if err != nil {
			log.Error(err, "Failed to create new Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
			return ctrl.Result{}, err
		}
	} else if err != nil {
		log.Error(err, "Failed to get Service")
		return ctrl.Result{}, err
	}

	// Handle Ingress if enabled
	if n8n.Spec.Ingress != nil && n8n.Spec.Ingress.Enable {
		ingress := &networkingv1.Ingress{}
		err = r.Get(ctx, types.NamespacedName{Name: n8n.Name, Namespace: n8n.Namespace}, ingress)
		if err != nil && apierrors.IsNotFound(err) {
			ing := r.ingressForN8n(n8n)
			log.Info("Creating a new Ingress", "Ingress.Namespace", ing.Namespace, "Ingress.Name", ing.Name)
			err = r.Create(ctx, ing)
			if err != nil {
				log.Error(err, "Failed to create new Ingress", "Ingress.Namespace", ing.Namespace, "Ingress.Name", ing.Name)
				return ctrl.Result{}, err
			}
		} else if err != nil {
			log.Error(err, "Failed to get Ingress")
			return ctrl.Result{}, err
		}
	}

	// Handle HTTPRoute if enabled
	if n8n.Spec.HTTPRoute != nil && n8n.Spec.HTTPRoute.Enable {
		httpRoute := &gatewayv1.HTTPRoute{}
		err = r.Get(ctx, types.NamespacedName{Name: n8n.Name, Namespace: n8n.Namespace}, httpRoute)
		if err != nil && apierrors.IsNotFound(err) {
			route := r.httpRouteForN8n(n8n)
			log.Info("Creating a new HTTPRoute", "HTTPRoute.Namespace", route.Namespace, "HTTPRoute.Name", route.Name)
			err = r.Create(ctx, route)
			if err != nil {
				log.Error(err, "Failed to create new HTTPRoute", "HTTPRoute.Namespace", route.Namespace, "HTTPRoute.Name", route.Name)
				return ctrl.Result{}, err
			}
		} else if err != nil {
			log.Error(err, "Failed to get HTTPRoute")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
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
					Host: n8n.Spec.Ingress.Hostname,
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
				gatewayv1.Hostname(n8n.Spec.HTTPRoute.Hostname),
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

		// Create PVC if it doesn't exist
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
			return nil, err
		}

		// Create PVC if it doesn't exist
		existingPvc := &corev1.PersistentVolumeClaim{}
		err := r.Get(context.TODO(), types.NamespacedName{Name: pvc.Name, Namespace: pvc.Namespace}, existingPvc)
		if err != nil && apierrors.IsNotFound(err) {
			if err := r.Create(context.TODO(), pvc); err != nil {
				return nil, err
			}
		} else if err != nil {
			return nil, err
		}
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
					Volumes: volumes,
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
							ContainerPort: 80,
							Name:          "http",
						}},
						Command: []string{"tini", "--", "/docker-entrypoint.sh"},
						Env: []corev1.EnvVar{
							{
								Name:  "DB_TYPE",
								Value: "postgresdb",
							},
							{
								Name:  "DB_POSTGRESDB_HOST",
								Value: n8n.Spec.Database.Postgres.Host,
							},
							{
								Name:  "DB_POSTGRESDB_PORT",
								Value: fmt.Sprintf("%d", n8n.Spec.Database.Postgres.Port),
							},
							{
								Name:  "DB_POSTGRESDB_DATABASE",
								Value: n8n.Spec.Database.Postgres.Database,
							},
							{
								Name:  "DB_POSTGRESDB_USER",
								Value: n8n.Spec.Database.Postgres.User,
							},
							{
								Name:  "DB_POSTGRESDB_PASSWORD",
								Value: n8n.Spec.Database.Postgres.Password,
							},
							{
								Name:  "DB_POSTGRESDB_SSL_REJECT_UNAUTHORIZED",
								Value: fmt.Sprintf("%t", !n8n.Spec.Database.Postgres.Ssl),
							},
						},
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
