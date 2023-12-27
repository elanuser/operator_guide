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
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	chartsv1 "github.com/example/bookstore-operator/api/v1"
)

// BookStoreReconciler reconciles a BookStore object
type BookStoreReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

var log = logf.Log.WithName("controller_bookstore")

// +kubebuilder:rbac:groups=charts.example.com,resources=bookstores,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=charts.example.com,resources=bookstores/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=charts.example.com,resources=bookstores/finalizers,verbs=update
func (r *BookStoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	reqLogger := log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling BookStore")

	// fetch the latest status of bookStore
	bookStore := chartsv1.BookStore{}
	if err := r.Client.Get(ctx, req.NamespacedName, &bookStore); err != nil {
		// Request object not found, could have been deleted after reconcile request.
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		reqLogger.Error(err, "Failed to retrieve the bookstore object")
		return reconcile.Result{}, err
	}

	err := r.ReconcileBookStore(ctx, &bookStore)
	if err != nil {
		reqLogger.Error(err, "Failed to reconcile the bookstore resources")
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BookStoreReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chartsv1.BookStore{}).
		Complete(r)
}

func (r *BookStoreReconciler) ReconcileBookStore(ctx context.Context, bookstore *chartsv1.BookStore) error {

	reqLogger := log.WithValues("Namespace", bookstore.Namespace)

	// Create/Update the bookStore deployment
	bookStoreDeploy := r.getBookStoreDeploy(bookstore)
	controllerutil.SetControllerReference(bookstore, bookStoreDeploy, r.Scheme)
	bookStoreDeployInCluster := &appsv1.Deployment{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: "bookstore", Namespace: bookstore.Namespace}, bookStoreDeployInCluster)
	if err != nil {
		reqLogger.Info("failed to retrieve the bookstore deployment", "error", err.Error(), "notFound", errors.IsNotFound(err))
		if !errors.IsNotFound(err) {
			return err
		}
		reqLogger.Info("start to create the bookstore deployment")
		if err = r.Client.Create(ctx, bookStoreDeploy); err != nil {
			return err
		}
	} else if !reflect.DeepEqual(bookStoreDeploy.Spec, bookStoreDeployInCluster.Spec) {
		bookStoreDeploy.ObjectMeta = bookStoreDeployInCluster.ObjectMeta
		err = r.Client.Update(ctx, bookStoreDeploy)
		if err != nil {
			return err
		}
		reqLogger.Info("bookstore deployment updated")
	}

	// Create/Update the bookStore service
	bookStoreSvc := r.getBookStoreService(bookstore)
	controllerutil.SetControllerReference(bookstore, bookStoreSvc, r.Scheme)
	bookStoreSvcInCluster := &corev1.Service{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: "bookstore-svc", Namespace: bookstore.Namespace}, bookStoreSvcInCluster)
	if err != nil {
		reqLogger.Info("failed to retrieve bookstore service", "IsNotFound", errors.IsNotFound(err))
		if !errors.IsNotFound(err) {
			return err
		}
		reqLogger.Info("start to create the bookstore service")
		if err = r.Client.Create(ctx, bookStoreSvc); err != nil {
			return err
		}
	} else if !reflect.DeepEqual(bookStoreSvc.Spec, bookStoreSvcInCluster.Spec) {
		bookStoreSvc.ObjectMeta = bookStoreSvcInCluster.ObjectMeta
		bookStoreSvc.Spec.ClusterIP = bookStoreSvcInCluster.Spec.ClusterIP
		if err = r.Client.Update(ctx, bookStoreSvc); err != nil {
			return err
		}
		reqLogger.Info("bookstore service updated")
	}

	// Create/Update the mongodb statefulSet
	mongoDBSts := r.getMongoDBStatefulSet(bookstore)
	controllerutil.SetControllerReference(bookstore, mongoDBSts, r.Scheme)
	mongoDBStsInCluster := &appsv1.StatefulSet{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: "mongodb", Namespace: bookstore.Namespace}, mongoDBStsInCluster)
	if err != nil {
		reqLogger.Info("failed to retrieve mongodb StatefulSet")
		if !errors.IsNotFound(err) {
			return err
		}
		reqLogger.Info("start to create the mongodb StatefulSet")
		if err = r.Client.Create(ctx, mongoDBSts); err != nil {
			return err
		}
		return err
	} else if !reflect.DeepEqual(mongoDBSts.Spec, mongoDBStsInCluster.Spec) {
		r.UpdateVolume(bookstore)
		mongoDBSts.ObjectMeta = mongoDBStsInCluster.ObjectMeta
		mongoDBSts.Spec.VolumeClaimTemplates = mongoDBStsInCluster.Spec.VolumeClaimTemplates
		if err = r.Client.Update(ctx, mongoDBSts); err != nil {
			return err
		}
		reqLogger.Info("mongodb StatefulSet updated")
	}

	// Create/Update the mongodb service
	mongoDBSvc := r.getMongoDBService(bookstore)
	controllerutil.SetControllerReference(bookstore, mongoDBSvc, r.Scheme)
	mongoDBSvcInCluster := &corev1.Service{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: "mongodb-service", Namespace: bookstore.Namespace}, mongoDBSvcInCluster)
	if err != nil {
		reqLogger.Info("failed to retrieve mongodb service")
		if !errors.IsNotFound(err) {
			return err
		}
		reqLogger.Info("start to create the mongodb service")
		if err = r.Client.Create(ctx, mongoDBSvc); err != nil {
			return err
		}
	} else if !reflect.DeepEqual(mongoDBSvc.Spec, mongoDBSvcInCluster.Spec) {
		mongoDBSvc.ObjectMeta = mongoDBSvcInCluster.ObjectMeta
		err = r.Client.Update(ctx, mongoDBSvc)
		if err != nil {
			return err
		}
		reqLogger.Info("mongodb-service updated")
	}

	r.Client.Status().Update(ctx, bookstore)
	return nil
}

func (r *BookStoreReconciler) getBookStoreService(bookstore *chartsv1.BookStore) *corev1.Service {

	var labels = map[string]string{"app": "bookstore"}

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "bookstore-svc",
			Namespace: bookstore.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name:       "tcp-port",
				Port:       bookstore.Spec.BookApp.Port,
				TargetPort: intstr.FromInt(bookstore.Spec.BookApp.TargetPort),
			}},
			Type:     bookstore.Spec.BookApp.ServiceType,
			Selector: labels,
		},
	}
}

func (r *BookStoreReconciler) getMongoDBService(bookstore *chartsv1.BookStore) *corev1.Service {

	var labels = map[string]string{"app": "bookstore-mongodb"}

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mongodb-service",
			Namespace: bookstore.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name: "tcp-port",
				Port: bookstore.Spec.BookDB.Port,
			}},
			Selector:  labels,
			ClusterIP: "None",
		},
	}
}

func (r *BookStoreReconciler) getBookStoreDeploy(bookstore *chartsv1.BookStore) *appsv1.Deployment {

	var labels = map[string]string{"app": "bookstore"}

	podTempSpec := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:            "bookstore",
				Image:           bookstore.Spec.BookApp.Repository + ":" + bookstore.Spec.BookApp.Tag,
				ImagePullPolicy: bookstore.Spec.BookApp.ImagePullPolicy,
			}},
		},
	}

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "bookstore",
			Namespace: bookstore.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Replicas: &bookstore.Spec.BookApp.Replicas,
			Template: podTempSpec,
		},
	}
}

func (r *BookStoreReconciler) getMongoDBStatefulSet(bookstore *chartsv1.BookStore) *appsv1.StatefulSet {

	var labels = map[string]string{"app": "bookstore-mongodb"}

	podTempSpec := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:            "mongodb",
				Image:           bookstore.Spec.BookDB.Repository + ":" + bookstore.Spec.BookDB.Tag,
				ImagePullPolicy: bookstore.Spec.BookDB.ImagePullPolicy,
			},
			}},
	}

	return &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StatefulSet",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mongodb",
			Namespace: bookstore.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "bookstore-mongodb"},
			},
			Replicas:             &bookstore.Spec.BookDB.Replicas,
			Template:             podTempSpec,
			ServiceName:          "mongodb-service",
			VolumeClaimTemplates: r.volClaimTemplate(bookstore.Spec.BookDB.StorageClass, bookstore.Spec.BookDB.DBSize),
		},
	}
}

func (r *BookStoreReconciler) volClaimTemplate(StorageClass string, DBSize resource.Quantity) []corev1.PersistentVolumeClaim {

	return []corev1.PersistentVolumeClaim{{
		ObjectMeta: metav1.ObjectMeta{
			Name: "mongodb-pvc",
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					//corev1.ResourceStorage: resource.MustParse(DBSize),
					corev1.ResourceStorage: DBSize,
				},
			},
			StorageClassName: &StorageClass,
		},
	}}
}

func (r *BookStoreReconciler) UpdateVolume(bookstore *chartsv1.BookStore) error {

	reqLogger := log.WithValues("Namespace", bookstore.Namespace)
	ctx := context.TODO()

	pvc := &corev1.PersistentVolumeClaim{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: "mongodb-pvc-mongodb-0", Namespace: bookstore.Namespace}, pvc); err != nil {
		return nil
	}

	if pvc.Spec.Resources.Requests[corev1.ResourceStorage] != bookstore.Spec.BookDB.DBSize {
		reqLogger.Info("Need to expand the mongodb volume")
		pvc.Spec.Resources.Requests[corev1.ResourceStorage] = bookstore.Spec.BookDB.DBSize
		err := r.Client.Update(ctx, pvc)
		if err != nil {
			reqLogger.Info("Error in expanding the mongodb volume")
			return err
		}
		reqLogger.Info("mongodb volume updated successfully")
	}
	return nil
}
