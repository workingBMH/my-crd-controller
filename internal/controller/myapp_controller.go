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

package controller

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	batchv1 "tutorial.kubebuilder.io/myapp/api/v1"
)

const myFinalizerName = "myapps.batch.tutorial.kubebuilder.io"

// MyAppReconciler reconciles a MyApp object
type MyAppReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

//+kubebuilder:rbac:groups=batch.tutorial.kubebuilder.io,resources=myapps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch.tutorial.kubebuilder.io,resources=myapps/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=batch.tutorial.kubebuilder.io,resources=myapps/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MyApp object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.0/pkg/reconcile
func (r *MyAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("myapp", req.NamespacedName)

	myApp := &batchv1.MyApp{}
	if err := r.Get(ctx, req.NamespacedName, myApp); err != nil {
		if errors.IsNotFound(err) {
			// 对象已被删除，返回并不尝试重新入队
			log.Info("MyApp resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// 出现其他错误时重新入队
		log.Error(err, "Failed to get MyApp")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 检查是否 MyApp 被标记为删除，这可以通过 DeletionTimestamp 属性是否被设置来判断
	if myApp.ObjectMeta.DeletionTimestamp.IsZero() {
		// MyApp 不处于删除过程中
		// 如果没有设置 finalizer，则添加 finalizer 并更新对象
		if !containsString(myApp.GetFinalizers(), myFinalizerName) {
			controllerutil.AddFinalizer(myApp, myFinalizerName)
			err := r.Update(ctx, myApp)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// MyApp 处于删除过程中
		if containsString(myApp.GetFinalizers(), myFinalizerName) {
			// 在这里执行 finalizer 逻辑
			// 如果清理逻辑成功，则从对象中移除 finalizer
			if err := r.finalizeMyApp(ctx, log, myApp); err != nil {
				// 如果无法执行清理逻辑，则返回错误
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(myApp, myFinalizerName)
			err := r.Update(ctx, myApp)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		// 停止重新入队，因为它已经被删除
		return ctrl.Result{}, nil
	}

	deploymentInfo := myApp.Spec.DeploymentTemplate

	// deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        deploymentInfo.ObjectMeta.Name,
			Namespace:   deploymentInfo.ObjectMeta.Namespace,
			Labels:      deploymentInfo.ObjectMeta.Labels,
			Annotations: deploymentInfo.ObjectMeta.Annotations,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &deploymentInfo.Spec.Replicas,
			Selector: &deploymentInfo.Spec.Selector,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: deploymentInfo.Spec.Template.Labels,
				},
				Spec: corev1.PodSpec{
					Volumes:      deploymentInfo.Spec.Template.Volumes,
					Containers:   []corev1.Container{},
					NodeSelector: deploymentInfo.Spec.Template.NodeSelector,
				},
			},
		},
	}

	for index := range deploymentInfo.Spec.Template.Containers {
		container := corev1.Container{
			Name:            deploymentInfo.Spec.Template.Containers[index].Name,
			Image:           deploymentInfo.Spec.Template.Containers[index].Image,
			Ports:           deploymentInfo.Spec.Template.Containers[index].Ports,
			VolumeMounts:    deploymentInfo.Spec.Template.Containers[index].VolumeMounts,
			Stdin:           deploymentInfo.Spec.Template.Containers[index].Stdin,
			StdinOnce:       deploymentInfo.Spec.Template.Containers[index].StdinOnce,
			TTY:             deploymentInfo.Spec.Template.Containers[index].TTY,
			Env:             deploymentInfo.Spec.Template.Containers[index].Envs,
			Resources:       deploymentInfo.Spec.Template.Containers[index].Resources,
			Args:            deploymentInfo.Spec.Template.Containers[index].Args,
			Command:         deploymentInfo.Spec.Template.Containers[index].Command,
			ImagePullPolicy: corev1.PullIfNotPresent,
		}
		deployment.Spec.Template.Spec.Containers = append(deployment.Spec.Template.Spec.Containers, container)
	}

	opResult, err := ctrl.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		// 设置 owner reference，这样 Deployment 将由 MyApp 自动管理
		if err := ctrl.SetControllerReference(myApp, deployment, r.Scheme); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		log.Error(err, "unable to create or update Deployment for MyApp", "deployment", deployment)
		return ctrl.Result{}, err
	}

	if opResult != controllerutil.OperationResultNone {
		log.Info("Deployment successfully reconciled", "operation", opResult)
	}

	// configmap
	configmap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        deploymentInfo.ObjectMeta.Name,
			Namespace:   deploymentInfo.ObjectMeta.Namespace,
			Labels:      deploymentInfo.ObjectMeta.Labels,
			Annotations: deploymentInfo.ObjectMeta.Annotations,
		},
		Data: map[string]string{},
	}

	for index := range deploymentInfo.Spec.Template.Containers {
		name := deploymentInfo.Spec.Template.Containers[index].Name
		config := deploymentInfo.Spec.Template.Containers[index].Config
		if config != "" {
			configmap.Data[name] = config
		}
	}

	if len(configmap.Data) != 0 {
		opResult, err = ctrl.CreateOrUpdate(ctx, r.Client, configmap, func() error {
			// 设置 owner reference，这样 ConfigMap 将由 MyApp 自动管理
			if err := ctrl.SetControllerReference(myApp, configmap, r.Scheme); err != nil {
				return err
			}
			return nil
		})

		if err != nil {
			log.Error(err, "unable to create or update configmap for MyApp", "configmap", configmap)
			return ctrl.Result{}, err
		}

		if opResult != controllerutil.OperationResultNone {
			log.Info("configmap successfully reconciled", "operation", opResult)
		}
	} else {
		// 检查集群中是否存在 ConfigMap
		foundConfigMap := &corev1.ConfigMap{}
		err := r.Get(ctx, types.NamespacedName{Name: deploymentInfo.ObjectMeta.Name, Namespace: deploymentInfo.ObjectMeta.Namespace}, foundConfigMap)
		if err != nil && errors.IsNotFound(err) {
			log.Info("ConfigMap not found and no data provided, nothing to do")
			// ConfigMap 不存在，没有任何操作需要执行
		} else if err != nil {
			log.Error(err, "Failed to get ConfigMap for MyApp")
			return ctrl.Result{}, err
		} else {
			// ConfigMap 存在，尝试删除它
			log.Info("Deleting existing ConfigMap as no data provided for it")
			err = r.Delete(ctx, foundConfigMap)
			if err != nil {
				log.Error(err, "Failed to delete ConfigMap for MyApp")
				return ctrl.Result{}, err
			}
			// 记录删除成功信息
			log.Info("ConfigMap deleted successfully")
		}
	}

	// update myApp status
	if err := r.Status().Update(ctx, myApp); err != nil {
		log.Error(err, "unable to update MyApp status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MyAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&batchv1.MyApp{}).
		Complete(r)
}

// finalizeMyApp 是一个负责清理外部资源的函数
func (r *MyAppReconciler) finalizeMyApp(ctx context.Context, log logr.Logger, myApp *batchv1.MyApp) error {
	// 清理逻辑，例如删除关联的 Deployment 和 ConfigMap

	// // 清理 Deployment
	// deployment := &appsv1.Deployment{
	// 	ObjectMeta: metav1.ObjectMeta{
	// 		Name:      myApp.Name,
	// 		Namespace: myApp.Namespace,
	// 	},
	// }
	// if err := r.Delete(ctx, deployment, client.PropagationPolicy(metav1.DeletePropagationForeground)); client.IgnoreNotFound(err) != nil {
	// 	log.Error(err, "Failed to delete Deployment during finalization", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
	// 	return err
	// }

	// 清理 ConfigMap
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      myApp.Name,
			Namespace: myApp.Namespace,
		},
	}

	// 使用预期的删除选项删除 ConfigMap
	if err := r.Delete(ctx, configMap, client.PropagationPolicy(metav1.DeletePropagationForeground)); client.IgnoreNotFound(err) != nil {
		log.Error(err, "Failed to delete ConfigMap during finalization", "ConfigMap.Namespace", configMap.Namespace, "ConfigMap.Name", configMap.Name)
		return err
	}

	log.Info("Successfully finalized MyApp")
	return nil
}

// containsString 是一个帮助函数，用于检查字符串切片中是否包含一个特定的字符串
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
