package controllers

import (
	"context"
	"strings"

	unistorev1alpha1 "github.com/itroyano/ukv-operator/api/v1alpha1"
	"github.com/itroyano/ukv-operator/controllers/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type VolumeToMount struct {
	Name      string
	ClaimName string
	MountPath string
	Owner     string
}

var volumeList []VolumeToMount

func (r *UKVReconciler) reconcileVolumesForUKV(ctx context.Context, ukvResource *unistorev1alpha1.UKV) error {
	logger := log.FromContext(ctx)
	for _, volume := range ukvResource.Spec.Volumes {
		mountName := strings.ReplaceAll(volume.MountPath, "/", "-")
		name := ukvResource.Name + mountName + "-volume"
		if err := r.getOrCreatePersistence(ctx, name, volume, ukvResource); err != nil {
			logger.Error(err, "Failed to reconcile PVC")
			return err
		}
	}
	return nil
}

func (r *UKVReconciler) getOrCreatePersistence(ctx context.Context, name string, vol unistorev1alpha1.Persistence, ukvResource *unistorev1alpha1.UKV) error {
	logger := log.FromContext(ctx)
	foundPvc := &corev1.PersistentVolumeClaim{}
	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: ukvResource.Namespace}, foundPvc)
	if err != nil && errors.IsNotFound(err) {
		// create a PVC
		logger.Info("Creating a new PVC", "Namespace", ukvResource.Namespace, "Name", name)
		pvcmode := corev1.PersistentVolumeFilesystem
		pvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: utils.SetObjectMeta(name, ukvResource.Namespace, map[string]string{}),
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
		// Set ukv instance as the owner and controller
		if err := ctrl.SetControllerReference(ukvResource, pvc, r.Scheme); err != nil {
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
	listedVolume := VolumeToMount{
		Name:      name,
		ClaimName: name,
		MountPath: vol.MountPath,
		Owner:     ukvResource.Name,
	}
	if !containsVolume(volumeList, listedVolume) {
		volumeList = append(volumeList, listedVolume)
	}

	return nil
}

func (r *UKVReconciler) GetVolumeList() []VolumeToMount {
	return volumeList
}

func containsVolume(slice []VolumeToMount, element VolumeToMount) bool {
	for _, a := range slice {
		if a == element {
			return true
		}
	}
	return false
}
