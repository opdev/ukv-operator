package controllers

import (
	"strconv"

	unistorev1alpha1 "github.com/itroyano/ukv-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// deploymentForUKV returns a UKV Deployment object
func (r *UKVReconciler) deploymentForUKV(ukvResource *unistorev1alpha1.UKV) *appsv1.Deployment {
	labels := labelsForUKV(ukvResource.Name)
	replicas := ukvResource.Spec.NumOfInstances
	resourceRequests := corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("200m"),
		corev1.ResourceMemory: resource.MustParse("100m"),
	}
	resourceLimits := corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse(ukvResource.Spec.ConcurrencyLimit),
		corev1.ResourceMemory: resource.MustParse(ukvResource.Spec.MemoryLimit),
	}
	deployment := &appsv1.Deployment{
		ObjectMeta: SetObjectMeta(ukvResource.Name, ukvResource.Namespace, map[string]string{}),
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
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "config",
							MountPath: "/var/lib/ukv/" + ukvResource.Spec.DBType,
						}},
						Resources: corev1.ResourceRequirements{
							Limits:   resourceLimits,
							Requests: resourceRequests,
						},
						Env: []corev1.EnvVar{
							{
								Name:  "dir",
								Value: "/var/lib/ukv/" + ukvResource.Spec.DBType,
							},
							{
								Name:  "port",
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

func getUKVImage(ukvResource *unistorev1alpha1.UKV) string {
	// TODO: conditions based on ukvResource.Spec.DBType.  for now just return latest umem for testing.
	return "docker.io/unum/ukv:latest"
}
