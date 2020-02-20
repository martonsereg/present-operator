/*

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
	"k8s.io/apimachinery/pkg/util/intstr"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	examplev1alpha1 "presentation/api/v1alpha1"
)

var labels = map[string]string{
	"app": "presentation",
}

// PresentationReconciler reconciles a Presentation object
type PresentationReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=example.meetup.com,resources=presentations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=example.meetup.com,resources=presentations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="apps",resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services;configmaps;pods,verbs=get;list;watch;create;update;patch;delete

func (r *PresentationReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("presentation", req.NamespacedName)

	config := &examplev1alpha1.Presentation{}
	err := r.Get(context.TODO(), req.NamespacedName, config)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	err = r.reconcileObjects(config)
	if err != nil {
		return reconcile.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *PresentationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&examplev1alpha1.Presentation{}).
		Owns(&appsv1.Deployment{}).
		Owns(&apiv1.ConfigMap{}).
		Owns(&apiv1.Service{}).
		Complete(r)
}

func (r *PresentationReconciler) reconcileObjects(config *examplev1alpha1.Presentation) error {
	desiredConfigMap := r.configMap(config.Spec.Content)
	desiredConfigMap.Namespace = "default"
	isController := true
	desiredConfigMap.OwnerReferences = []metav1.OwnerReference{
		{
			Kind:       config.Kind,
			APIVersion: config.APIVersion,
			Name:       config.Name,
			UID:        config.GetUID(),
			Controller: &isController,
		},
	}
	key, err := client.ObjectKeyFromObject(desiredConfigMap)
	if err != nil {
		return err
	}

	currentConfigMap := &apiv1.ConfigMap{}

	err = r.Get(context.TODO(), key, currentConfigMap)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if apierrors.IsNotFound(err) {
		if err := r.Create(context.TODO(), desiredConfigMap); err != nil {
			r.Log.Error(err, "failed to reconcile config map")
			return err
		}
	} else {
		desiredConfigMap.ResourceVersion = currentConfigMap.ResourceVersion
		if !reflect.DeepEqual(desiredConfigMap, currentConfigMap) {
			if err := r.Update(context.TODO(), desiredConfigMap); err != nil {
				return err
			}
			if !reflect.DeepEqual(desiredConfigMap.Data, currentConfigMap.Data) {
				pods := &apiv1.PodList{}
				err := r.List(context.TODO(), pods, client.MatchingLabels(labels))
				if err != nil {
					return err
				}
				r.Log.Info("found pods", "len", len(pods.Items))
				for _, pod := range pods.Items {
					err = r.Delete(context.TODO(), &pod)
					if err != nil {
						r.Log.Error(err, "failed to delete pods")
						return err
					}
				}
				r.Log.Info("deleted pods in deployment")
			}
		} else {
			r.Log.Info("no changes in configmap")
		}
	}

	r.Log.Info("configmap reconciled")

	desiredDeployment := r.deployment(int32(config.Spec.Replicas))
	desiredDeployment.OwnerReferences = []metav1.OwnerReference{
		{
			Kind:       config.Kind,
			APIVersion: config.APIVersion,
			Name:       config.Name,
			UID:        config.GetUID(),
			Controller: &isController,
		},
	}
	key, err = client.ObjectKeyFromObject(desiredDeployment)
	if err != nil {
		return err
	}

	currentDeployment := &appsv1.Deployment{}

	err = r.Get(context.TODO(), key, currentDeployment)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if apierrors.IsNotFound(err) {
		if err := r.Create(context.TODO(), desiredDeployment); err != nil {
			r.Log.Error(err, "failed to reconcile deployment")
			return err
		}
	} else {
		desiredDeployment.ResourceVersion = currentDeployment.ResourceVersion
		if !reflect.DeepEqual(desiredDeployment, currentDeployment) {
			if err := r.Update(context.TODO(), desiredDeployment); err != nil {
				return err
			}
		} else {
			r.Log.Info("no changes in deployment")
		}
	}

	r.Log.Info("deployment reconciled")

	desiredService := r.service(int32(config.Spec.ServicePort))
	desiredService.OwnerReferences = []metav1.OwnerReference{
		{
			Kind:       config.Kind,
			APIVersion: config.APIVersion,
			Name:       config.Name,
			UID:        config.GetUID(),
			Controller: &isController,
		},
	}
	key, err = client.ObjectKeyFromObject(desiredService)
	if err != nil {
		return err
	}

	currentService := &apiv1.Service{}

	err = r.Get(context.TODO(), key, currentService)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if apierrors.IsNotFound(err) {
		if err := r.Create(context.TODO(), desiredService); err != nil {
			r.Log.Error(err, "failed to reconcile service")
			return err
		}
	} else {
		desiredService.ResourceVersion = currentService.ResourceVersion
		desiredService.Spec.ClusterIP = currentService.Spec.ClusterIP
		if !reflect.DeepEqual(desiredService, currentService) {
			if err := r.Update(context.TODO(), desiredService); err != nil {
				return err
			}
		} else {
			r.Log.Info("no changes in service")
		}
	}

	r.Log.Info("service reconciled")

	return nil
}

func (r *PresentationReconciler) configMap(content string) *apiv1.ConfigMap {
	return &apiv1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "present-content",
			Namespace: "default",
			Labels:    labels,
		},
		Data: map[string]string{
			"meetup.slide": content,
		},
	}
}

func (r *PresentationReconciler) deployment(replicas int32) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "present",
			Namespace: "default",
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:  "present",
							Image: "mkboudreau/go-present:latest",
							Ports: []apiv1.ContainerPort{
								{
									ContainerPort: 3999,
									Protocol:      apiv1.ProtocolTCP,
								},
							},
							VolumeMounts: []apiv1.VolumeMount{
								{
									Name:      "content",
									MountPath: "/app",
									ReadOnly:  false,
								},
							},
						},
					},
					Volumes: []apiv1.Volume{
						{
							Name: "content",
							VolumeSource: apiv1.VolumeSource{
								ConfigMap: &apiv1.ConfigMapVolumeSource{
									LocalObjectReference: apiv1.LocalObjectReference{
										Name: "present-content",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *PresentationReconciler) service(port int32) *apiv1.Service {
	return &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "present",
			Namespace: "default",
			Labels:    labels,
		},
		Spec: apiv1.ServiceSpec{
			Ports: []apiv1.ServicePort{
				{
					Name:       "http-slides",
					Port:       port,
					TargetPort: intstr.FromInt(3999),
					Protocol:   apiv1.ProtocolTCP,
				},
			},
			Selector: labels,
			Type:     apiv1.ServiceTypeClusterIP,
		},
	}
}
