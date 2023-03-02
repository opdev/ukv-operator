/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	unistorev1alpha1 "github.com/itroyano/ukv-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UKVReconciler reconciles a UKV object
type UKVReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=unistore.unum.cloud,resources=ukvs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=unistore.unum.cloud,resources=ukvs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=unistore.unum.cloud,resources=ukvs/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the UKV object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *UKVReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var logger = log.FromContext(ctx)

	var ukvResource unistorev1alpha1.UKV

	if err := r.Get(ctx, req.NamespacedName, &ukvResource); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			logger.Info("UKV resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		logger.Error(err, "Failed to get UKV resource")
		return ctrl.Result{}, err
	}

	if errorDep := r.reconcileDeployment(ctx, &ukvResource); errorDep != nil {
		return ctrl.Result{}, errorDep
	}
	if errSvc := r.reconcileService(ctx, &ukvResource); errSvc != nil {
		return ctrl.Result{}, errSvc
	}
	//	if err = r.updateUKVStatus(ctx, &ukvResource); err != nil {
	//		return ctrl.Result{}, err
	//	}

	return ctrl.Result{}, nil
}

func (r *UKVReconciler) reconcileDeployment(ctx context.Context, ukvResource *unistorev1alpha1.UKV) error {
	var logger = log.FromContext(ctx)
	found := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: ukvResource.Name, Namespace: ukvResource.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// A new deployment needs to be created
		desiredDeployment := r.deploymentForUKV(ukvResource)
		logger.Info("Creating a new Deployment", "Deployment.Namespace", desiredDeployment.Namespace, "Deployment.Name", desiredDeployment.Name)
		err = r.Create(ctx, desiredDeployment)
		if err != nil {
			logger.Error(err, "Failed to create new Deployment", "Deployment.Namespace", desiredDeployment.Namespace, "Deployment.Name", desiredDeployment.Name)
			return err
		}
	}
	// TODO: implement r.Update(ctx, found) logic for ensuring the desired state is equal to current state

	return nil
}

func (r *UKVReconciler) reconcileService(ctx context.Context, ukvResource *unistorev1alpha1.UKV) error {
	var logger = log.FromContext(ctx)
	foundSvc := &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{Name: ukvResource.Spec.DBServiceName, Namespace: ukvResource.Namespace}, foundSvc)
	if err != nil && errors.IsNotFound(err) {
		// A new service needs to be created
		desiredService := r.serviceForUKV(ukvResource)
		logger.Info("Creating a new Service", "Service.Namespace", desiredService.Namespace, "Service.Name", desiredService.Name)
		err = r.Create(ctx, desiredService)
		if err != nil {
			logger.Error(err, "Failed to create new Service", "Service.Namespace", desiredService.Namespace, "Service.Name", desiredService.Name)
			return err
		}
	}

	// TODO: implement r.Update(ctx, found) logic for ensuring the desired state is equal to current state

	return nil

}

func (r *UKVReconciler) updateUKVStatus(ctx context.Context, ukvResource *unistorev1alpha1.UKV) error {
	// TODO: status logic for success & failure
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *UKVReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&unistorev1alpha1.UKV{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}

// labelsForUKV returns the labels for selecting the resources
// belonging to the given UKV resource name.
func labelsForUKV(name string) map[string]string {
	return map[string]string{"app": "ukv", "ownerInstance": name}
}

func SetObjectMeta(name string, namespace string, labels map[string]string) metav1.ObjectMeta {
	objectMeta := metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
		Labels:    labels,
	}
	return objectMeta
}
