package controllers

import (
	"context"
	"strings"

	unumv1alpha1 "github.com/opdev/ustore-operator/api/v1alpha1"
	"github.com/opdev/ustore-operator/controllers/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type volumeToMount struct {
	Name      string
	ClaimName string
	MountPath string
	Owner     string
}

var volumeList []volumeToMount

func (r *UStoreReconciler) reconcileVolumesForUStore(ctx context.Context, ustoreResource *unumv1alpha1.UStore) error {
	logger := log.FromContext(ctx)
	for _, volume := range ustoreResource.Spec.Volumes {
		mountName := strings.ReplaceAll(volume.MountPath, "/", "-")
		name := ustoreResource.Name + mountName + "-volume"
		if err := r.getOrCreatePersistence(ctx, name, volume, ustoreResource); err != nil {
			logger.Error(err, "Failed to reconcile PVC")
			return err
		}
	}
	return nil
}

func (r *UStoreReconciler) getOrCreatePersistence(ctx context.Context, name string, vol unumv1alpha1.Persistence, ustoreResource *unumv1alpha1.UStore) error {
	logger := log.FromContext(ctx)
	foundPvc := &corev1.PersistentVolumeClaim{}
	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: ustoreResource.Namespace}, foundPvc)
	if err != nil && errors.IsNotFound(err) {
		// create a PVC
		logger.Info("Creating a new PVC", "Namespace", ustoreResource.Namespace, "Name", name)
		pvcmode := corev1.PersistentVolumeFilesystem
		pvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: utils.SetObjectMeta(name, ustoreResource.Namespace, utils.LabelsForUStore(ustoreResource.Name)),
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.PersistentVolumeAccessMode(vol.AccessMode)},
				VolumeMode:  &pvcmode,
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						"storage": resource.MustParse(vol.Size),
					},
				},
			},
		}
		// Set ustore instance as the owner and controller
		if err := ctrl.SetControllerReference(ustoreResource, pvc, r.Scheme); err != nil {
			logger.Error(err, "Failed to set owner reference on PVC", name)
			return err
		}

		// create in k8s
		err := r.Create(ctx, pvc)
		if err != nil {
			logger.Error(err, "Failed to create PVC", name)
			return err
		}
	}
	listedVolume := volumeToMount{
		Name:      name,
		ClaimName: name,
		MountPath: vol.MountPath,
		Owner:     ustoreResource.Name,
	}
	if !containsVolume(volumeList, listedVolume) {
		volumeList = append(volumeList, listedVolume)
	}

	return nil
}

func (r *UStoreReconciler) getVolumeList() []volumeToMount {
	return volumeList
}

func containsVolume(slice []volumeToMount, element volumeToMount) bool {
	for _, a := range slice {
		if a == element {
			return true
		}
	}
	return false
}
