package controllers

import (
	"context"
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
	deployment := &appsv1.Deployment{
		ObjectMeta: utils.SetObjectMeta(ustoreResource.Name, ustoreResource.Namespace, map[string]string{}),
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: getUStoreImage(ustoreResource),
						Name:  "ustore",
						Command: []string{
							"./" + ustoreResource.Spec.DBType + "_server",
						},
						Args: []string{
							"--config",
							"$(DBCONFIG)",
							"--port",
							"$(DBPORT)",
						},
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "config",
							MountPath: "/var/lib/ustore/" + ustoreResource.Spec.DBType + "/",
						}},
						Resources: corev1.ResourceRequirements{
							Limits:   resourceLimits,
							Requests: resourceRequests,
						},
						Env: []corev1.EnvVar{
							{
								Name:  "DBCONFIG",
								Value: "/var/lib/ustore/" + ustoreResource.Spec.DBType + "/config.json",
							},
							{
								Name:  "DBPORT",
								Value: strconv.Itoa(ustoreResource.Spec.DBServicePort),
							},
						},
					}},
					Volumes: []corev1.Volume{{
						Name: "config",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: ustoreResource.Spec.DBConfigMapName,
								},
							},
						},
					}},
				},
			},
		},
	}
	r.addVolumesIfNeeded(deployment, ustoreResource)
	r.addAffinityIfNeeded(deployment, ustoreResource)
	// Set UStore instance as the owner and controller
	ctrl.SetControllerReference(ustoreResource, deployment, r.Scheme)
	return deployment
}

func (r *UStoreReconciler) addVolumesIfNeeded(deployment *appsv1.Deployment, ustoreResource *unumv1alpha1.UStore) {
	for _, volumeMount := range r.GetVolumeList() {
		if volumeMount.Owner == ustoreResource.Name {
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
			deployment.Spec.Template.Spec.Containers[0].VolumeMounts = append(deployment.Spec.Template.Spec.Containers[0].VolumeMounts, containerMount)
			deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, volume)
		}
	}
}

func (r *UStoreReconciler) addAffinityIfNeeded(deployment *appsv1.Deployment, ustoreResource *unumv1alpha1.UStore) {
	if len(ustoreResource.Spec.NodeAffinityLabels) > 0 {

		preferredSchedulingTerms := []corev1.PreferredSchedulingTerm{}

		for _, labelKeyValue := range ustoreResource.Spec.NodeAffinityLabels {
			term := corev1.PreferredSchedulingTerm{
				Weight: labelKeyValue.Weight,
				Preference: corev1.NodeSelectorTerm{
					MatchExpressions: []corev1.NodeSelectorRequirement{
						{
							Key:      labelKeyValue.Label,
							Operator: corev1.NodeSelectorOpIn,
							Values:   []string{labelKeyValue.Value},
						},
					},
				},
			}
			preferredSchedulingTerms = append(preferredSchedulingTerms, term)
		}

		deployment.Spec.Template.Spec.Affinity = &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				PreferredDuringSchedulingIgnoredDuringExecution: preferredSchedulingTerms,
			},
		}
	}
}

func getUStoreImage(ustoreResource *unumv1alpha1.UStore) string {
	// TODO: conditions based on ustoreResource.Spec.DBType.  for now just return latest ucset for testing.
	return "quay.io/gurgen_yegoryan/ustore:0.12.1"
}
