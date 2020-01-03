package rbdcomponent

import (
	"context"

	"github.com/go-logr/logr"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
)

type controllerForRainbond func(p *rainbondv1alpha1.RbdComponent) interface{}

var log = logf.Log.WithName("controller_rbdcomponent")

// Add creates a new RbdComponent Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileRbdComponent{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("rbdcomponent-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource RbdComponent
	err = c.Watch(&source.Kind{Type: &rainbondv1alpha1.RbdComponent{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource DaemonSet and requeue the owner RbdComponent
	err = c.Watch(&source.Kind{Type: &appv1.DaemonSet{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &rainbondv1alpha1.RbdComponent{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileRbdComponent implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileRbdComponent{}

// ReconcileRbdComponent reconciles a RbdComponent object
type ReconcileRbdComponent struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a RbdComponent object and makes changes based on the state read
// and what is in the RbdComponent.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileRbdComponent) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling RbdComponent")

	// Fetch the RbdComponent instance
	instance := &rainbondv1alpha1.RbdComponent{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if instance.Name == "rbd-package" {
		if !r.isRbdHubReady() {
			reqLogger.Info("rbd-hub is not ready", "component", instance.Name)
			return reconcile.Result{Requeue: true}, err
		}

		if err := handleRainbondPackage("/opt/rainbond/pkg/rainbond-pkg-V5.2-dev.tgz", "/opt/rainbond/pkg"); err != nil {
			reqLogger.Error(err, "handle rainbond package")
			return reconcile.Result{Requeue: true}, nil

		}
		return reconcile.Result{}, nil
	}

	if instance.Name == "rbd-etcd" {
		generics := []interface{}{
			podForEtcd0(instance),
			serviceForEtcd0(instance),
		}
		for _, generic := range generics {
			// Set PrivateRegistry instance as the owner and controller
			if err := controllerutil.SetControllerReference(instance, generic.(metav1.Object), r.scheme); err != nil {
				return reconcile.Result{}, err
			}

			// Check if the statefulset already exists, if not create a new one
			reconcileResult, err := r.updateOrCreateResource(reqLogger, generic.(runtime.Object), generic.(metav1.Object))
			if err != nil {
				return reconcileResult, err
			}
		}
	}

	// Install image repository
	if instance.Name == "rbd-hub" {
		reqLogger.Info("Reconciling rbd-hub")
		generics := []interface{}{
			r.daemonSetForHub(instance),
			r.serviceForHub(instance),
			r.persistentVolumeClaimForHub(instance),
			ingressForHub(instance),
			secretForHub(instance),
		}
		for _, generic := range generics {
			// Set PrivateRegistry instance as the owner and controller
			if err := controllerutil.SetControllerReference(instance, generic.(metav1.Object), r.scheme); err != nil {
				return reconcile.Result{}, err
			}

			// Check if the statefulset already exists, if not create a new one
			reconcileResult, err := r.updateOrCreateResource(reqLogger, generic.(runtime.Object), generic.(metav1.Object))
			if err != nil {
				return reconcileResult, err
			}
		}
	}

	if instance.Name == "rbd-gateway" {
		generic := daemonSetForGateway(instance)

		// Set PrivateRegistry instance as the owner and controller
		if err := controllerutil.SetControllerReference(instance, generic.(metav1.Object), r.scheme); err != nil {
			return reconcile.Result{}, err
		}

		// Check if the statefulset already exists, if not create a new one
		reconcileResult, err := r.updateOrCreateResource(reqLogger, generic.(runtime.Object), generic.(metav1.Object))
		if err != nil {
			return reconcileResult, err
		}
	}

	if instance.Name == "rbd-db" {
		reqLogger.Info("Reconciling rbd-hub")
		generics := []interface{}{
			statefulsetForRainbondDB(instance),
			serviceForDB(instance),
		}
		for _, generic := range generics {
			// Set PrivateRegistry instance as the owner and controller
			if err := controllerutil.SetControllerReference(instance, generic.(metav1.Object), r.scheme); err != nil {
				return reconcile.Result{}, err
			}

			// Check if the statefulset already exists, if not create a new one
			reconcileResult, err := r.updateOrCreateResource(reqLogger, generic.(runtime.Object), generic.(metav1.Object))
			if err != nil {
				return reconcileResult, err
			}
		}
	}

	controllerForRainbondFuncs := map[string]controllerForRainbond{
		"rbd-app-ui":   deploymentForRainbondAppUI,
		"rbd-worker":   daemonSetForRainbondWorker,
		"rbd-api":      daemonSetForRainbondAPI,
		"rbd-chaos":    daemonSetForRainbondChaos,
		"rbd-eventlog": daemonSetForRainbondEventlog,
		"rbd-monitor":  daemonSetForRainbondMonitor,
		"rbd-mq":       daemonSetForRainbondMQ,
		"rbd-dns":      daemonSetForRainbondDNS,
	}
	for name := range controllerForRainbondFuncs {
		generic := controllerForRainbondFuncs[name](instance)
		reqLogger.Info("Name", name, "Reconciling", generic.(runtime.Object).GetObjectKind().GroupVersionKind().Kind)
		// Set PrivateRegistry instance as the owner and controller
		if err := controllerutil.SetControllerReference(instance, generic.(metav1.Object), r.scheme); err != nil {
			return reconcile.Result{}, err
		}

		// Check if the statefulset already exists, if not create a new one
		reconcileResult, err := r.updateOrCreateResource(reqLogger, generic.(runtime.Object), generic.(metav1.Object))
		if err != nil {
			return reconcileResult, err
		}
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileRbdComponent) updateOrCreateResource(reqLogger logr.Logger, obj runtime.Object, meta metav1.Object) (reconcile.Result, error) {
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: meta.GetName(), Namespace: meta.GetNamespace()}, obj)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new", obj.GetObjectKind().GroupVersionKind().Kind, "Namespace", meta.GetNamespace(), "Name", meta.GetName())
		err = r.client.Create(context.TODO(), obj)
		if err != nil {
			reqLogger.Error(err, "Failed to create new", obj.GetObjectKind(), "Namespace", meta.GetNamespace(), "Name", meta.GetName())
			return reconcile.Result{}, err
		}
		// daemonset created successfully - return and requeue TODO: why?
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "Failed to get ", obj.GetObjectKind())
		return reconcile.Result{}, err
	}

	// obj exsits, update
	reqLogger.Info("Update ", obj.GetObjectKind().GroupVersionKind().Kind, "Namespace", meta.GetNamespace(), "Name", meta.GetName())
	if err := r.client.Update(context.TODO(), obj); err != nil {
		reqLogger.Error(err, "Failed to update ", obj.GetObjectKind())
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

// labelsForRbdComponent returns the labels for selecting the resources
// belonging to the given PrivateRegistry CR name.
func labelsForRbdComponent(name string) map[string]string {
	return map[string]string{"name": name} // TODO: only one rainbond?
}

func (r *ReconcileRbdComponent) isRbdHubReady() bool {
	reqLogger := log.WithName("Check if rbd-hub is ready")

	eps := &corev1.EndpointsList{}
	listOpts := []client.ListOption{
		client.MatchingLabels(map[string]string{
			"name": "rbd-hub",
		}),
	}
	err := r.client.List(context.TODO(), eps, listOpts...)
	if err != nil {
		reqLogger.Error(err, "list rbd-hub endpints")
		return false
	}

	for _, ep := range eps.Items {
		for _, subset := range ep.Subsets {
			if len(subset.Addresses) > 0 {
				reqLogger.Info("Found a healthy endpoint address", "address", subset.Addresses[0])
				return true
			}
		}
	}

	return false
}
