package controllers

import (
	"context"
	"strconv"

	"github.com/imdario/mergo"
	unistorev1alpha1 "github.com/itroyano/ukv-operator/api/v1alpha1"
	"github.com/itroyano/ukv-operator/controllers/utils"
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

func (r *UKVReconciler) reconcileDeployment(ctx context.Context, ukvResource *unistorev1alpha1.UKV) error {
	logger := log.FromContext(ctx)
	found := &appsv1.Deployment{}
	desiredDeployment := r.deploymentForUKV(ukvResource)
	err := r.Get(ctx, types.NamespacedName{Name: ukvResource.Name, Namespace: ukvResource.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// A new deployment needs to be created
		logger.Info("Creating a new Deployment", "Deployment.Namespace", desiredDeployment.Namespace, "Deployment.Name", desiredDeployment.Name)
		err = r.Create(ctx, desiredDeployment)
		if err != nil {
			logger.Error(err, "Failed to create new Deployment", "Deployment.Namespace", desiredDeployment.Namespace, "Deployment.Name", desiredDeployment.Name)
			ukvResource.Status.DeploymentStatus = "Failed Creation"
			_ = r.Status().Update(ctx, ukvResource)
			return err
		}
		// update status for deployment
		ukvResource.Status.DeploymentName = desiredDeployment.Name
		ukvResource.Status.DeploymentStatus = "Successful"
		err := r.Status().Update(ctx, ukvResource)
		if err != nil {
			logger.Error(err, "Failed to update UKV Deployment status")
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
	ukvResource.Status.DeploymentName = desiredDeployment.Name
	ukvResource.Status.DeploymentStatus = "Successful"
	err = r.Status().Update(ctx, ukvResource)
	if err != nil {
		logger.Error(err, "Failed to update UKV Deployment status")
		return err
	}
	return nil
}

// deploymentForUKV returns a UKV Deployment object
func (r *UKVReconciler) deploymentForUKV(ukvResource *unistorev1alpha1.UKV) *appsv1.Deployment {
	labels := utils.LabelsForUKV(ukvResource.Name)
	replicas := ukvResource.Spec.NumOfInstances
	resourceRequests := corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("200m"),
		corev1.ResourceMemory: resource.MustParse("100Mi"),
	}
	resourceLimits := corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse(ukvResource.Spec.ConcurrencyLimit),
		corev1.ResourceMemory: resource.MustParse(ukvResource.Spec.MemoryLimit),
	}
	deployment := &appsv1.Deployment{
		ObjectMeta: utils.SetObjectMeta(ukvResource.Name, ukvResource.Namespace, map[string]string{}),
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
						Image: getUKVImage(ukvResource),
						Name:  "ukv",
						Command: []string{
							"./" + ukvResource.Spec.DBType + "_server",
						},
						Args: []string{
							"--config",
							"$(DBCONFIG)",
							"--port",
							"$(DBPORT)",
						},
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "config",
							MountPath: "/var/lib/ukv/" + ukvResource.Spec.DBType + "/",
						}},
						Resources: corev1.ResourceRequirements{
							Limits:   resourceLimits,
							Requests: resourceRequests,
						},
						Env: []corev1.EnvVar{
							{
								Name:  "DBCONFIG",
								Value: "/var/lib/ukv/" + ukvResource.Spec.DBType + "/config.json",
							},
							{
								Name:  "DBPORT",
								Value: strconv.Itoa(ukvResource.Spec.DBServicePort),
							},
						},
					}},
					Volumes: []corev1.Volume{{
						Name: "config",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: ukvResource.Spec.DBConfigMapName,
								},
							},
						},
					}},
				},
			},
		},
	}
	r.addVolumesIfNeeded(deployment, ukvResource)
	// Set UKV instance as the owner and controller
	ctrl.SetControllerReference(ukvResource, deployment, r.Scheme)
	return deployment
}

func (r *UKVReconciler) addVolumesIfNeeded(deployment *appsv1.Deployment, ukvResource *unistorev1alpha1.UKV) {
	for _, volumeMount := range r.GetVolumeList() {
		if volumeMount.Owner == ukvResource.Name {
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

func getUKVImage(ukvResource *unistorev1alpha1.UKV) string {
	// TODO: conditions based on ukvResource.Spec.DBType.  for now just return latest umem for testing.
	return "docker.io/unum/ukv:latest"
}
