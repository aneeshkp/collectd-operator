package collectd

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	collectdmonv1alpha1 "github.com/aneeshkp/collectd-operator/pkg/apis/collectdmon/v1alpha1"
	"github.com/aneeshkp/collectd-operator/pkg/resources/configmaps"
	"github.com/aneeshkp/collectd-operator/pkg/resources/deployments"
	"github.com/aneeshkp/collectd-operator/pkg/resources/serviceaccounts"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_collectd")

//ReturnValues ...
type ReturnValues struct {
	hash256String string
	reQueue       bool
	err           error
}

const maxConditions = 6

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Collectd Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconciappsle.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileCollectd{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("collectd-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Collectd
	err = c.Watch(&source.Kind{Type: &collectdmonv1alpha1.Collectd{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for configmap
	err = c.Watch(&source.Kind{Type: &corev1.ConfigMap{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &collectdmonv1alpha1.Collectd{},
	})

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Collectd
	//err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
	//	IsController: true,
	//	OwnerType:    &collectdmonv1alpha1.Collectd{},
	//})
	//if err != nil {
	//	return err
	//}

	// Watch for daemonset
	err = c.Watch(&source.Kind{Type: &appsv1.DaemonSet{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &collectdmonv1alpha1.Collectd{},
	})

	// Watch for changes to secondary resource ServiceAccount and requeue the owner Interconnect
	err = c.Watch(&source.Kind{Type: &corev1.ServiceAccount{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &collectdmonv1alpha1.Collectd{},
	})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Collectd
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &collectdmonv1alpha1.Collectd{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileCollectd{}

// ReconcileCollectd reconciles a Collectd object
type ReconcileCollectd struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Collectd object and makes changes based on the state read
// and what is in the Collectd.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileCollectd) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Collectd")

	// Fetch the Collectd instance
	instance := &collectdmonv1alpha1.Collectd{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Info("Request object not found")
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Assign the generated resource version to the status
	if instance.Status.RevNumber == "" {
		instance.Status.RevNumber = instance.ObjectMeta.ResourceVersion
		r.UpdateCondition(instance, "provision spec to desired state", reqLogger)
	}

	// Check if serviceaccount already exists, if not create a new one
	returnValues := r.ReconcileServiceAccount(instance, reqLogger)
	if returnValues.err != nil {
		return reconcile.Result{}, err
	} else if returnValues.reQueue {
		return reconcile.Result{Requeue: true}, nil
	}

	//currentConfigHash, err := r.ReconcileConfigMap(instance, reqLogger)
	returnValues = r.ReconcileConfigMapWithHash(instance, reqLogger)
	if returnValues.err != nil {
		return reconcile.Result{}, err
	} else if returnValues.reQueue {
		return reconcile.Result{Requeue: true}, nil
	}

	//desiredConfigMap := &corev1.ConfigMap{} // where to ge desired configmap
	//eq := reflect.DeepEqual(currentConfigMap, currentConfigMap)
	//currentConfigHash
	returnValues = r.ReconcileDeployment(instance, returnValues.hash256String, reqLogger)
	if returnValues.err != nil {
		return reconcile.Result{}, err
	} else if returnValues.reQueue {
		return reconcile.Result{Requeue: true}, nil
	}

	//size := instance.Spec.DeploymentPlan.Size

	// Pod already exists - don't requeue

	return reconcile.Result{}, nil
}

//UpdateCondition ...
func (r *ReconcileCollectd) UpdateCondition(instance *collectdmonv1alpha1.Collectd, reason string, reqLogger logr.Logger) error {
	// update status
	// update status
	condition := collectdmonv1alpha1.CollectdCondition{
		Type:           collectdmonv1alpha1.CollectdConditionProvisioning,
		Reason:         reason,
		TransitionTime: metav1.Now(),
	}
	instance.Status.Conditions = addCondition(instance.Status.Conditions, condition)
	r.client.Status().Update(context.TODO(), instance)
	return nil
}

//ReconcileServiceAccount  ...
func (r *ReconcileCollectd) ReconcileServiceAccount(instance *collectdmonv1alpha1.Collectd, reqLogger logr.Logger) ReturnValues {
	svcaccnt := serviceaccounts.NewServiceAccountForCR(instance)
	// Set OutgoingPortal instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, svcaccnt, r.scheme); err != nil {
		return ReturnValues{"", false, err}
	}
	svcaccntFound := &corev1.ServiceAccount{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: svcaccnt.Name, Namespace: svcaccnt.Namespace}, svcaccntFound)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new ServiceAccount", "svcaccnt.Namespace", svcaccnt.Namespace, "svcaccnt.Name", svcaccnt.Name)
		err = r.client.Create(context.TODO(), svcaccnt)
		if err != nil {
			reqLogger.Error(err, "Failed to create new ServiceAccount")
			return ReturnValues{"", false, err}
		}
		//Success , Requeue
		return ReturnValues{"", true, err}
	} else if err != nil {
		reqLogger.Error(err, "Failed to get ServiceAccount")
		return ReturnValues{"", false, err}
	}
	// Service Account already exists - don't requeue
	reqLogger.Info("Skip reconcile: Service Account already exists", "svcaccnt.Namespace",
		svcaccntFound.Namespace, "svcaccnt.Name", svcaccntFound.Name)
	return ReturnValues{"", false, nil}
}

func addCondition(conditions []collectdmonv1alpha1.CollectdCondition, condition collectdmonv1alpha1.CollectdCondition) []collectdmonv1alpha1.CollectdCondition {
	size := len(conditions) + 1
	first := 0
	if size > maxConditions {
		first = size - maxConditions
	}
	return append(conditions, condition)[first:size]
}

//ReconcileConfigMapWithHash  ../
func (r *ReconcileCollectd) ReconcileConfigMapWithHash(instance *collectdmonv1alpha1.Collectd, reqLogger logr.Logger) ReturnValues {
	// This is done little differently .
	configMapFound := &corev1.ConfigMap{}
	//Check if config map is defined
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Spec.DeploymentPlan.Configname, Namespace: instance.Namespace}, configMapFound)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("ConfigMap not found... Use default Config\n")

		configmap := configmaps.NewConfigMapForCR(instance)
		if err := controllerutil.SetControllerReference(instance, configmap, r.scheme); err != nil {
			reqLogger.Info("ERROR createing owner to config")
			return ReturnValues{"", false, err}
		}
		err = r.client.Create(context.TODO(), configmap)
		if err != nil {
			reqLogger.Info("ERROR createing config")
			return ReturnValues{"", false, err}
		}

		return ReturnValues{"", true, nil}

	} else if err != nil {
		reqLogger.Error(err, "Error loading  configmap\n")
		return ReturnValues{"", false, err}
	}

	//Calculate hash for existing configdata
	out, err := json.Marshal(configMapFound)
	if err != nil {
		reqLogger.Error(err, "Failed to calculate hash\n")
		return ReturnValues{"", false, err}
	}
	h := sha256.New()
	h.Write(out)
	currentConfigHash := fmt.Sprintf("%x", h.Sum(nil))
	return ReturnValues{currentConfigHash, false, nil}
}

//ReconcileDeployment  ...
func (r *ReconcileCollectd) ReconcileDeployment(instance *collectdmonv1alpha1.Collectd, currentConfigHash string, reqLogger logr.Logger) ReturnValues {
	// Define a new deployment
	dep := deployments.NewDaemonSetForCR(instance)
	if err := controllerutil.SetControllerReference(instance, dep, r.scheme); err != nil {
		return ReturnValues{"", false, err}
	}
	//check if deployment already exists
	//depFound := &appsv1.Deployment{}
	depFound := &appsv1.DaemonSet{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: dep.Name, Namespace: dep.Namespace}, depFound)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		// Define a new deployment
		if dep.Annotations == nil {
			dep.Annotations = make(map[string]string)
		}
		dep.Annotations["configHash"] = currentConfigHash

		reqLogger.Info("Creating a new Deployment", "Daemonset", dep)
		err = r.client.Create(context.TODO(), dep)
		if err != nil {
			return ReturnValues{"", false, err}
		}
		r.UpdateCondition(instance, "Daemonset Created", reqLogger)
		return ReturnValues{"", true, nil}
	} else if err != nil {
		reqLogger.Error(err, "Failed to get Daemonset\n")
		return ReturnValues{"", false, err}
	}
	// Deployment already exists - don't requeue
	reqLogger.Info("Skip reconcile: Deployment already exists", "Deployment.Namespace", depFound.Namespace, "Deployment.Name", depFound.Name)
	deployedConfigHash := depFound.Annotations["configHash"]

	if deployedConfigHash != currentConfigHash {
		r.UpdateCondition(instance, "Configuration changed", reqLogger)
		reqLogger.Info("Change in configMap , delete deployment.")
		err = r.client.Delete(context.TODO(), depFound)
		if err != nil {
			reqLogger.Error(err, "Failed to update deployment")
			return ReturnValues{"", false, err}
		}
		r.UpdateCondition(instance, "Deleteing daemonset for config updates", reqLogger)
		return ReturnValues{"", true, nil}
	}
	return ReturnValues{"", false, nil}
}
