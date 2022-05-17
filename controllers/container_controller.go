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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
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
//+kubebuilder:rbac:groups=core,resources=services,verbs=list;watch;get;patch;create;update
//+kubebuilder:rbac:groups=core,resources=services/status,verbs=get

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
	container := workloadsv1.Container{}
	err := r.Get(ctx, req.NamespacedName, &container)
	if err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	log.Info("r.Get", "App.Namespace", container.Namespace, "App.Name", container.Name)

	deployment, err := r.newDeployment(container)
	if err != nil {
		log.Error(err, "newDeployment failed")

		return ctrl.Result{}, err
	}

	log.Info("newDeployment", "App.Namespace", container.Namespace, "App.Name", container.Name)

	service, err := r.newService(container)
	if err != nil {
		return ctrl.Result{}, err
	}
	log.Info("newService", "App.Namespace", container.Namespace, "App.Name", container.Name)

	ingress, err := r.newIngress(container)
	if err != nil {
		return ctrl.Result{}, err
	}

	log.Info("newIngress", "App.Namespace", container.Namespace, "App.Name", container.Name)

	applyOpts := []client.PatchOption{client.ForceOwnership, client.FieldOwner(service.Name + "-controller")}

	err = r.Patch(ctx, &deployment, client.Apply, applyOpts...)
	if err != nil {
		return ctrl.Result{}, err
	}
	log.Info("r.Patch -deployment", "App.Namespace", container.Namespace, "App.Name", container.Name)

	err = r.Patch(ctx, &service, client.Apply, applyOpts...)
	if err != nil {
		return ctrl.Result{}, err
	}
	log.Info("r.Patch -service", "App.Namespace", container.Namespace, "App.Name", container.Name)

	err = r.Patch(ctx, &ingress, client.Apply, applyOpts...)
	if err != nil {
		return ctrl.Result{}, err
	}
	log.Info("r.Patch -ingress", "App.Namespace", container.Namespace, "App.Name", container.Name)

	found := &appsv1.Deployment{}

	err = r.Get(ctx, types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating Deployment", "deployment", deployment.Name)

		err = r.Create(ctx, &deployment)
		if err != nil {
			log.Error(err, "unable to create Deployment", "deployment", deployment.Name)

			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	if found.Spec.Replicas != deployment.Spec.Replicas {
		found.Spec.Replicas = deployment.Spec.Replicas
		log.Info("Updating Deployment", "deployment", deployment.Name)

		err = r.Update(ctx, found)
		if err != nil {
			log.Error(err, "unable to update Deployment", "deployment", deployment.Name)

			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *ContainerReconciler) newDeployment(container workloadsv1.Container) (appsv1.Deployment, error) {
	deployment := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      container.Name + "-deployment",
			Namespace: container.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: container.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": container.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": container.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "app",
							Image: container.Spec.Image,
						},
					},
				},
			},
		},
	}

	err := controllerutil.SetControllerReference(&container, &deployment, r.Scheme)
	if err != nil {
		return deployment, err
	}

	return deployment, nil
}

func (r *ContainerReconciler) newService(container workloadsv1.Container) (corev1.Service, error) {
	service := corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      container.Name + "-service",
			Namespace: container.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   "TCP",
					Port:       8080,
					TargetPort: intstr.FromInt(80),
				},
			},
			Selector: map[string]string{
				"app": container.Name,
			},
		},
	}

	err := controllerutil.SetControllerReference(&container, &service, r.Scheme)
	if err != nil {
		return service, err
	}

	return service, nil
}

func (r *ContainerReconciler) newIngress(container workloadsv1.Container) (netv1.Ingress, error) {
	pathTypePrefix := netv1.PathTypePrefix
	ingress := netv1.Ingress{
		TypeMeta: metav1.TypeMeta{
			APIVersion: netv1.SchemeGroupVersion.String(),
			Kind:       "Ingress",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      container.Name + "-ingress",
			Namespace: container.Namespace,
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/use-regex": "true",
				"cert-manager.io/cluster-issuer":        container.Spec.ClusterIssuer,
			},
		},
		Spec: netv1.IngressSpec{
			Rules: []netv1.IngressRule{
				{
					Host: container.Spec.Host,
					IngressRuleValue: netv1.IngressRuleValue{
						HTTP: &netv1.HTTPIngressRuleValue{
							Paths: []netv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathTypePrefix,
									Backend: netv1.IngressBackend{
										Service: &netv1.IngressServiceBackend{
											Name: container.Name + "-service",
											Port: netv1.ServiceBackendPort{
												Number: 8080,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			TLS: []netv1.IngressTLS{
				{
					Hosts: []string{
						container.Spec.Host,
					},
					SecretName: container.Name + "-tls",
				},
			},
		},
	}

	err := controllerutil.SetControllerReference(&container, &ingress, r.Scheme)
	if err != nil {
		return ingress, err
	}

	return ingress, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ContainerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&workloadsv1.Container{}).
		Owns(&corev1.Service{}).
		Owns(&appsv1.Deployment{}).
		Owns(&netv1.Ingress{}).
		Complete(r)
}
