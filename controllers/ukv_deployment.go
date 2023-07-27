package controllers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/imdario/mergo"
	unumv1alpha1 "github.com/opdev/ustore-operator/api/v1alpha1"
	"github.com/opdev/ustore-operator/controllers/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *UStoreReconciler) reconcileDeployment(ctx context.Context, ustoreResource *unumv1alpha1.UStore) error {
	logger := log.FromContext(ctx)
	found := &appsv1.Deployment{}
	desiredDeployment := r.deploymentForUStore(ustoreResource)
	err := r.Get(ctx, types.NamespacedName{Name: ustoreResource.Name, Namespace: ustoreResource.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// A new deployment needs to be created
		logger.Info("Creating a new Deployment", "Deployment.Namespace", desiredDeployment.Namespace, "Deployment.Name", desiredDeployment.Name)
		err = r.Create(ctx, desiredDeployment)
		if err != nil {
			logger.Error(err, "Failed to create new Deployment", "Deployment.Namespace", desiredDeployment.Namespace, "Deployment.Name", desiredDeployment.Name)
			ustoreResource.Status.DeploymentStatus = "Failed Creation"
			_ = r.Status().Update(ctx, ustoreResource)
			return err
		}
		// update status for deployment
		ustoreResource.Status.DeploymentName = desiredDeployment.Name
		ustoreResource.Status.DeploymentStatus = "Successful"
		err := r.Status().Update(ctx, ustoreResource)
		if err != nil {
			logger.Error(err, "Failed to update UStore Deployment status")
			return err
		}
		return nil
	}

	// patch only if there is a difference between desired and current.
	patchDiff := client.MergeFrom(found.DeepCopyObject().(client.Object))
	if err := mergo.Merge(found, desiredDeployment, mergo.WithOverride); err != nil {
		logger.Error(err, "Error in merge")
		return err
	}

	if err := r.Patch(ctx, found, patchDiff); err != nil {
		logger.Error(err, "Failed to update Deployment to desired state", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
		return err
	}

	// update status for deployment
	ustoreResource.Status.DeploymentName = desiredDeployment.Name
	ustoreResource.Status.DeploymentStatus = "Successful"
	err = r.Status().Update(ctx, ustoreResource)
	if err != nil {
		logger.Error(err, "Failed to update UStore Deployment status")
		return err
	}

	return nil
}

// deploymentForUStore returns a UStore Deployment object
func (r *UStoreReconciler) deploymentForUStore(ustoreResource *unumv1alpha1.UStore) *appsv1.Deployment {
	labels := utils.LabelsForUStore(ustoreResource.Name)
	replicas := ustoreResource.Spec.NumOfInstances
	resourceRequests := corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("200m"),
		corev1.ResourceMemory: resource.MustParse("100Mi"),
	}
	resourceLimits := corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse(ustoreResource.Spec.ConcurrencyLimit),
		corev1.ResourceMemory: resource.MustParse(ustoreResource.Spec.MemoryLimit),
	}

	volumes := []corev1.Volume{
		{
			Name: ustore_config_name,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: ustoreResource.Spec.DBConfigMapName,
					},
				},
			},
		},
	}

	volumeMounts := []corev1.VolumeMount{
		{
			Name:      ustore_config_name,
			MountPath: fmt.Sprintf("%s/%s/", ustore_workdir, ustoreResource.Spec.DBType),
		},
	}

	volumes, volumeMounts = r.addVolumesIfNeeded(ustoreResource, volumes, volumeMounts)

	containers := []corev1.Container{
		{
			Image:   getUStoreImage(ustoreResource),
			Name:    ustore_container_name,
			Command: []string{fmt.Sprintf("./%s_server", ustoreResource.Spec.DBType)},
			Args: []string{
				"--config",
				"$(DBCONFIG)",
				"--port",
				"$(DBPORT)",
			},
			VolumeMounts: volumeMounts,
			Resources: corev1.ResourceRequirements{
				Limits:   resourceLimits,
				Requests: resourceRequests,
			},
			Env: []corev1.EnvVar{
				{
					Name:  "DBCONFIG",
					Value: fmt.Sprintf("%s/%s/config.json", ustore_workdir, ustoreResource.Spec.DBType),
				},
				{
					Name:  "DBPORT",
					Value: strconv.Itoa(ustoreResource.Spec.DBServicePort),
				},
			},
		},
	}

	podTemplate := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: labels,
		},
		Spec: corev1.PodSpec{
			Containers: containers,
			Volumes:    volumes,
		},
	}

	deploymentSpec := appsv1.DeploymentSpec{
		Replicas: &replicas,
		Selector: &metav1.LabelSelector{
			MatchLabels: labels,
		},
		Template: podTemplate,
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: utils.SetObjectMeta(ustoreResource.Name, ustoreResource.Namespace, map[string]string{}),
		Spec:       deploymentSpec,
	}

	if affinity := r.addAffinityIfNeeded(ustoreResource); affinity != nil {
		deployment.Spec.Template.Spec.Affinity = affinity
	}

	if pullSecrets := r.addPullSecretRefsIfNeeded(ustoreResource); pullSecrets != nil {
		deployment.Spec.Template.Spec.ImagePullSecrets = pullSecrets
	}

	// Set UStore instance as the owner and controller
	ctrl.SetControllerReference(ustoreResource, deployment, r.Scheme)
	return deployment
}

func (r *UStoreReconciler) addVolumesIfNeeded(ustoreResource *unumv1alpha1.UStore, volumes []corev1.Volume, volumeMounts []corev1.VolumeMount) ([]corev1.Volume, []corev1.VolumeMount) {
	for _, volumeMount := range r.getVolumeList() {
		if volumeMount.Owner != ustoreResource.Name {
			continue
		}
		containerMount := corev1.VolumeMount{
			Name:      volumeMount.Name,
			MountPath: volumeMount.MountPath,
		}
		volume := corev1.Volume{
			Name: volumeMount.Name,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: volumeMount.ClaimName,
				},
			},
		}

		volumeMounts = append(volumeMounts, containerMount)
		volumes = append(volumes, volume)
	}

	return volumes, volumeMounts
}

func (r *UStoreReconciler) addAffinityIfNeeded(ustoreResource *unumv1alpha1.UStore) *corev1.Affinity {
	if len(ustoreResource.Spec.NodeAffinityLabels) <= 0 {
		return nil
	}
	preferredSchedulingTerms := []corev1.PreferredSchedulingTerm{}

	for _, labelKeyValue := range ustoreResource.Spec.NodeAffinityLabels {
		matchExpression := corev1.NodeSelectorRequirement{
			Key:      labelKeyValue.Label,
			Operator: corev1.NodeSelectorOpIn,
			Values:   []string{labelKeyValue.Value},
		}
		term := corev1.PreferredSchedulingTerm{
			Weight: labelKeyValue.Weight,
			Preference: corev1.NodeSelectorTerm{
				MatchExpressions: []corev1.NodeSelectorRequirement{matchExpression},
			},
		}
		preferredSchedulingTerms = append(preferredSchedulingTerms, term)
	}

	return &corev1.Affinity{
		NodeAffinity: &corev1.NodeAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: preferredSchedulingTerms,
		},
	}
}

func (r *UStoreReconciler) addPullSecretRefsIfNeeded(ustoreResource *unumv1alpha1.UStore) []corev1.LocalObjectReference {
	if ustoreResource.Spec.DBType != "udisk" {
		return nil
	}

	pullSecrets := []corev1.LocalObjectReference{}

	ee_pull_secret := corev1.LocalObjectReference{
		Name: ustore_ee_pull_secret,
	}

	pullSecrets = append(pullSecrets, ee_pull_secret)

	return pullSecrets
}

func getUStoreImage(ustoreResource *unumv1alpha1.UStore) string {
	if ustoreResource.Spec.DBType == "udisk" {
		return ustore_ee_image
	}
	return ustore_ce_image
}
