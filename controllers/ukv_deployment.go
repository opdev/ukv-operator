package controllers

import (
	unistorev1alpha1 "github.com/itroyano/ukv-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// deploymentForUKV returns a UKV Deployment object
func (r *UKVReconciler) deploymentForUKV(ukvResource *unistorev1alpha1.UKV) *appsv1.Deployment {
	labels := labelsForUKV(ukvResource.Name)
	replicas := int32(1) //TODO: parametrize this in CRD
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
					}},
				},
			},
		},
	}
	// Set UKV instance as the owner and controller
	ctrl.SetControllerReference(ukvResource, deployment, r.Scheme)
	return deployment

}

func getUKVImage(ukvResource *unistorev1alpha1.UKV) string {
	//TODO: conditions based on ukvResource.Spec.DBType.  for now just return latest umem for testing.
	return "docker.io/unum/ukv:latest"
}
