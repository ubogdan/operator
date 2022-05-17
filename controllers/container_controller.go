/*
Copyright 2022 Bogdan Ungureanu.

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

	appsV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logger "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	workloadsv1 "github.com/ubogdan/operator/api/v1"
)

// ContainerReconciler reconciles a Container object
type ContainerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=workloads.operator.io,resources=containers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=workloads.operator.io,resources=containers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=workloads.operator.io,resources=containers/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the Container object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *ContainerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logger.FromContext(ctx)

	// Fetch the Container instance
	container := &workloadsv1.Container{}
	err := r.Get(ctx, req.NamespacedName, container)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Container resource not found. Ignoring since object must be deleted.")
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}

		log.Info("Unable to fetch Container resource...")
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	log.Info("r.Get", "App.Namespace", container.Namespace, "App.Name", container.Name)

	deployment := appsV1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      container.Name + "-deployment",
			Namespace: container.Namespace,
		},
		Spec: appsV1.DeploymentSpec{
			Replicas: container.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": container.Name,
				},
			},
			Template: coreV1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": container.Name,
					},
				},
				Spec: coreV1.PodSpec{
					Containers: []coreV1.Container{
						{
							Name:  "app",
							Image: container.Spec.Image,
						},
					},
				},
			},
		},
	}

	log.Info("setControllerReference", "App.Namespace", container.Namespace, "App.Name", container.Name)

	err = controllerutil.SetControllerReference(container, &deployment, r.Scheme)
	if err != nil {
		return ctrl.Result{}, err
	}

	found := &appsV1.Deployment{}

	err = r.Get(ctx, types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		log.V(1).Info("Creating Deployment", "deployment", deployment.Name)

		err = r.Create(ctx, &deployment)
		if err != nil {
			log.Error(err, "unable to create Deployment", "deployment", deployment.Name)

			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	if found.Spec.Replicas != deployment.Spec.Replicas {
		found.Spec.Replicas = deployment.Spec.Replicas
		log.V(1).Info("Updating Deployment", "deployment", deployment.Name)

		err = r.Update(ctx, found)
		if err != nil {
			log.Error(err, "unable to update Deployment", "deployment", deployment.Name)

			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ContainerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&workloadsv1.Container{}).
		Owns(&appsV1.Deployment{}).
		Complete(r)
}
