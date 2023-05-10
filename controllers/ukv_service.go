package controllers

import (
	"context"
	"strconv"

	unumv1alpha1 "github.com/itroyano/ukv-operator/api/v1alpha1"
	"github.com/itroyano/ukv-operator/controllers/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	corev1 "k8s.io/api/core/v1"
)

func (r *UStoreReconciler) reconcileService(ctx context.Context, ustoreResource *unumv1alpha1.UStore) error {
	logger := log.FromContext(ctx)
	foundSvc := &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{Name: ustoreResource.Name, Namespace: ustoreResource.Namespace}, foundSvc)
	if err != nil && errors.IsNotFound(err) {
		// A new service needs to be created
		desiredService := r.serviceForUStore(ustoreResource)
		logger.Info("Creating a new Service", "Service.Namespace", desiredService.Namespace, "Service.Name", desiredService.Name)
		err = r.Create(ctx, desiredService)
		if err != nil {
			logger.Error(err, "Failed to create new Service", "Service.Namespace", desiredService.Namespace, "Service.Name", desiredService.Name)
			ustoreResource.Status.ServiceStatus = "Failed Creation"
			_ = r.Status().Update(ctx, ustoreResource)
			return err
		}
		// update status for service
		ustoreResource.Status.ServiceUrl = desiredService.Name + "." + desiredService.Namespace + ".svc.cluster.local" + ":" + strconv.Itoa(ustoreResource.Spec.DBServicePort)
		ustoreResource.Status.ServiceStatus = "Successful"
		err := r.Status().Update(ctx, ustoreResource)
		if err != nil {
			logger.Error(err, "Failed to update UStore Service status")
			return err
		}
		return nil // done creating a new service
	}

	if foundSvc.Spec.Ports[0].Port != int32(ustoreResource.Spec.DBServicePort) {
		foundSvc.Spec.Ports[0].Port = int32(ustoreResource.Spec.DBServicePort)
		foundSvc.Spec.Ports[0].TargetPort = intstr.FromInt(ustoreResource.Spec.DBServicePort)
		err := r.Update(ctx, foundSvc)
		if err != nil {
			logger.Error(err, "Failed to update UStore Service")
			ustoreResource.Status.ServiceStatus = "Failed"
			_ = r.Status().Update(ctx, ustoreResource)
			return err
		}
		// update the status to show the correct url
		ustoreResource.Status.ServiceUrl = foundSvc.Name + "." + foundSvc.Namespace + ".svc.cluster.local" + ":" + strconv.Itoa(ustoreResource.Spec.DBServicePort)
		ustoreResource.Status.ServiceStatus = "Successful"
		_ = r.Status().Update(ctx, ustoreResource)
	}

	return nil
}

// serviceForUStore returns a UStore Service object
func (r *UStoreReconciler) serviceForUStore(ustoreResource *unumv1alpha1.UStore) *corev1.Service {
	service := &corev1.Service{
		ObjectMeta: utils.SetObjectMeta(ustoreResource.Name, ustoreResource.Namespace, utils.LabelsForUStore(ustoreResource.Name)),
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name:       "db",
				Protocol:   corev1.ProtocolTCP,
				Port:       int32(ustoreResource.Spec.DBServicePort),
				TargetPort: intstr.FromInt(ustoreResource.Spec.DBServicePort),
			}},
			Selector: utils.LabelsForUStore(ustoreResource.Name),
			Type:     corev1.ServiceTypeClusterIP,
		},
	}
	// Set UStore instance as the owner and controller
	ctrl.SetControllerReference(ustoreResource, service, r.Scheme)
	return service
}
