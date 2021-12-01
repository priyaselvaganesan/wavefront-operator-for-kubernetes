/*
Copyright 2021.

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

package wavefrontcollector

import (
	"context"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	wavefrontcollectorv1alpha1 "github.com/wavefronthq/wavefront-operator-for-kubernetes/apis/wavefrontcollector/v1alpha1"
)

var log = logf.Log.WithName("controller_wavefrontcollector")

// WavefrontCollectorReconciler reconciles a WavefrontCollector object
type WavefrontCollectorReconciler struct {
	Client client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=wavefrontcollector.wavefront.com,resources=wavefrontcollectors,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=wavefrontcollector.wavefront.com,resources=wavefrontcollectors/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=wavefrontcollector.wavefront.com,resources=wavefrontcollectors/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the WavefrontCollector object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *WavefrontCollectorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	reqLogger := log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling WavefrontCollector")

	// Fetch the WavefrontCollector instance
	instance := &wavefrontv1alpha1.WavefrontCollector{}
	err := r.client.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile req.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the req.
		return ctrl.Result{}, err
	}

	desiredCRInstance := instance.DeepCopy()
	getLatestCollector(reqLogger, desiredCRInstance)

	var updateCR bool
	if instance.Spec.Daemon {
		updateCR, err = r.reconcileDaemonSet(reqLogger, desiredCRInstance)
	} else {
		updateCR, err = r.reconcileDeployment(reqLogger, desiredCRInstance)
	}
	if err != nil {
		return ctrl.Result{}, err
	} else if updateCR {
		err := r.updateCRStatus(reqLogger, instance, desiredCRInstance)
		if err != nil {
			reqLogger.Error(err, "Failed to update WavefrontCollector CR status")
			return ctrl.Result{}, err
		}
		reqLogger.Info("Updated WavefrontCollector CR Status.")
	}
	return ctrl.Result{RequeueAfter: 1 * time.Hour}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WavefrontCollectorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&wavefrontcollectorv1alpha1.WavefrontCollector{}).
		Complete(r)
}
