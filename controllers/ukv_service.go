package controllers

import (
	"context"
	"strconv"

	unistorev1alpha1 "github.com/itroyano/ukv-operator/api/v1alpha1"
	"github.com/itroyano/ukv-operator/controllers/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	corev1 "k8s.io/api/core/v1"
)

func (r *UKVReconciler) reconcileService(ctx context.Context, ukvResource *unistorev1alpha1.UKV) error {
	logger := log.FromContext(ctx)
	foundSvc := &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{Name: ukvResource.Name, Namespace: ukvResource.Namespace}, foundSvc)
	if err != nil && errors.IsNotFound(err) {
		// A new service needs to be created
		desiredService := r.serviceForUKV(ukvResource)
		logger.Info("Creating a new Service", "Service.Namespace", desiredService.Namespace, "Service.Name", desiredService.Name)
		err = r.Create(ctx, desiredService)
		if err != nil {
			logger.Error(err, "Failed to create new Service", "Service.Namespace", desiredService.Namespace, "Service.Name", desiredService.Name)
			ukvResource.Status.ServiceStatus = "Failed Creation"
			_ = r.Status().Update(ctx, ukvResource)
			return err
		}
		// update status for service
		ukvResource.Status.ServiceUrl = desiredService.Name + "." + desiredService.Namespace + ".svc.cluster.local" + ":" + strconv.Itoa(ukvResource.Spec.DBServicePort)
		ukvResource.Status.ServiceStatus = "Successful"
		err := r.Status().Update(ctx, ukvResource)
		if err != nil {
			logger.Error(err, "Failed to update UKV Service status")
			return err
		}
		return nil // done creating a new service
	}

	if foundSvc.Spec.Ports[0].Port != int32(ukvResource.Spec.DBServicePort) {
		foundSvc.Spec.Ports[0].Port = int32(ukvResource.Spec.DBServicePort)
		foundSvc.Spec.Ports[0].TargetPort = intstr.FromInt(ukvResource.Spec.DBServicePort)
		err := r.Update(ctx, foundSvc)
		if err != nil {
			logger.Error(err, "Failed to update UKV Service")
			ukvResource.Status.ServiceStatus = "Failed"
			_ = r.Status().Update(ctx, ukvResource)
			return err
		}
		// update the status to show the correct url
		ukvResource.Status.ServiceUrl = foundSvc.Name + "." + foundSvc.Namespace + ".svc.cluster.local" + ":" + strconv.Itoa(ukvResource.Spec.DBServicePort)
		ukvResource.Status.ServiceStatus = "Successful"
		_ = r.Status().Update(ctx, ukvResource)
	}

	return nil
}

// serviceForUKV returns a UKV Service object
func (r *UKVReconciler) serviceForUKV(ukvResource *unistorev1alpha1.UKV) *corev1.Service {
	service := &corev1.Service{
		ObjectMeta: utils.SetObjectMeta(ukvResource.Name, ukvResource.Namespace, utils.LabelsForUKV(ukvResource.Name)),
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name:       "db",
				Protocol:   corev1.ProtocolTCP,
				Port:       int32(ukvResource.Spec.DBServicePort),
				TargetPort: intstr.FromInt(ukvResource.Spec.DBServicePort),
			}},
			Selector: utils.LabelsForUKV(ukvResource.Name),
			Type:     corev1.ServiceTypeClusterIP,
		},
	}
	// Set UKV instance as the owner and controller
	ctrl.SetControllerReference(ukvResource, service, r.Scheme)
	return service
}
