package controller

import (
	"os"
	"strings"

	n8nv1alpha1 "github.com/jakub-k-slys/n8n-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
)

const defaultN8nVersion = "1.85.3"

// BuildTimeN8nVersion is set at build time via ldflags
var BuildTimeN8nVersion string

// getN8nVersion determines the n8n version to use
// Priority: 1) Build-time injected version, 2) .version file, 3) default constant
func getN8nVersion() string {
	// Use build-time injected version if available
	if BuildTimeN8nVersion != "" {
		return BuildTimeN8nVersion
	}

	// Try to read version from .version file
	if versionBytes, err := os.ReadFile(".version"); err == nil {
		version := strings.TrimSpace(string(versionBytes))
		if version != "" {
			return version
		}
	}

	// Fallback to default version
	return defaultN8nVersion
}

var n8nVersion = getN8nVersion()
var n8nDockerImage = "ghcr.io/n8n-io/n8n:" + n8nVersion

func labelsForN8n() map[string]string {
	var imageTag string
	image := n8nDockerImage
	imageTag = strings.Split(image, ":")[1]
	return map[string]string{
		"app.kubernetes.io/name":       "n8n-operator",
		"app.kubernetes.io/version":    imageTag,
		"app.kubernetes.io/managed-by": "N8nController",
	}
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
