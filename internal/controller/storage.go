package controller

import (
	"context"

	n8nv1alpha1 "github.com/jakub-k-slys/n8n-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

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
