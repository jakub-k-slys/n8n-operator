package controller

import (
	n8nv1alpha1 "github.com/jakub-k-slys/n8n-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *N8nReconciler) deploymentForN8n(n8n *n8nv1alpha1.N8n) (*appsv1.Deployment, error) {
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
