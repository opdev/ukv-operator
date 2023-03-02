package controllers

import (
	unistorev1alpha1 "github.com/itroyano/ukv-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"

	corev1 "k8s.io/api/core/v1"
)

// serviceForUKV returns a UKV Service object
func (r *UKVReconciler) serviceForUKV(ukvResource *unistorev1alpha1.UKV) *corev1.Service {
	service := &corev1.Service{
		ObjectMeta: SetObjectMeta(ukvResource.Spec.DBServiceName, ukvResource.Namespace, map[string]string{}),
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name:       "db",
				Protocol:   corev1.ProtocolTCP,
				Port:       int32(ukvResource.Spec.DBServicePort),
				TargetPort: intstr.FromInt(ukvResource.Spec.DBServicePort),
			}},
			Selector: labelsForUKV(ukvResource.Name),
			Type:     corev1.ServiceTypeClusterIP,
		},
	}
	// Set UKV instance as the owner and controller
	ctrl.SetControllerReference(ukvResource, service, r.Scheme)
	return service
}
