package controller

import (
	"fmt"

	n8nv1alpha1 "github.com/jakub-k-slys/n8n-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

const (
	defaultUserID  = 1000
	defaultGroupID = 1000
)

func getPodSecurityContext() *corev1.PodSecurityContext {
	return &corev1.PodSecurityContext{
		RunAsNonRoot: &[]bool{true}[0],
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
		FSGroup: &[]int64{defaultGroupID}[0],
	}
}

func getContainerSecurityContext() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		RunAsNonRoot:             &[]bool{true}[0],
		RunAsUser:                &[]int64{defaultUserID}[0],
		AllowPrivilegeEscalation: &[]bool{false}[0],
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{
				"ALL",
			},
		},
	}
}

func getN8nEnvVars(n8n *n8nv1alpha1.N8n) []corev1.EnvVar {
	return []corev1.EnvVar{
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
		{
			Name:  "N8N_USER_FOLDER",
			Value: "/home/node",
		},
		{
			Name:  "N8N_EDITOR_BASE_URL",
			Value: fmt.Sprintf("https://%s", n8n.Spec.Hostname.Url),
		},
		{
			Name:  "N8N_TEMPLATES_ENABLED",
			Value: "true",
		},
		{
			Name:  "N8N_HOST",
			Value: fmt.Sprintf("https://%s", n8n.Spec.Hostname.Url),
		},
		{
			Name:  "WEBHOOK_URL",
			Value: n8n.Spec.Hostname.Url,
		},
		{
			Name:  "N8N_METRICS",
			Value: fmt.Sprintf("%t", n8n.Spec.Metrics != nil && n8n.Spec.Metrics.Enable),
		},
	}
}
