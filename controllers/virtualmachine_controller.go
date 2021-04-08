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

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	vmv1alpha1 "github.com/tmax-cloud/hypercloud-ovirt-operator/api/v1alpha1"
	"github.com/tmax-cloud/hypercloud-ovirt-operator/pkg/ovirt"
)

const virtualMachineFinalizer = "vm.tmaxcloud.com/finalizer"

// VirtualMachineReconciler reconciles a VirtualMachine object
type VirtualMachineReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Actuator *ovirt.OvirtActuator
}

//+kubebuilder:rbac:groups=vm.tmaxcloud.com,resources=virtualmachines,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=vm.tmaxcloud.com,resources=virtualmachines/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=vm.tmaxcloud.com,resources=virtualmachines/finalizers,verbs=update

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *VirtualMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("virtualmachine", req.NamespacedName)
	reqLogger.Info("Reconciling VirtualMachine")

	vm := &vmv1alpha1.VirtualMachine{}
	err := r.Get(ctx, req.NamespacedName, vm)
	if err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Info("VirtualMachine resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		reqLogger.Error(err, "Failed to get VirtualMachine")
		return ctrl.Result{}, err
	}

	// Check if the VM instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isVmMarkedToBeDeleted := vm.GetDeletionTimestamp() != nil
	if isVmMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(vm, virtualMachineFinalizer) {
			// Run finalization logic for virtualMachineFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.Actuator.FinalizeVm(reqLogger, vm); err != nil {
				reqLogger.Error(err, "Failed to finalize VirtualMachine")
				return ctrl.Result{}, err
			}

			// Remove virtualMachineFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(vm, virtualMachineFinalizer)
			err := r.Update(ctx, vm)
			if err != nil {
				reqLogger.Error(err, "Failed to remove finalizer from VirtualMachine")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer for this CR
	if !controllerutil.ContainsFinalizer(vm, virtualMachineFinalizer) {
		controllerutil.AddFinalizer(vm, virtualMachineFinalizer)
		err = r.Update(ctx, vm)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	// Check if the VirtualMachine already exists, if not create a new one
	err = r.Actuator.GetVM(reqLogger, vm)
	if err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Info("Creating a new VirtualMachine", "vm.Name", vm.Name)
			err = r.Actuator.AddVM(reqLogger, vm)
			if err != nil {
				reqLogger.Error(err, "Failed to create new VirtualMachine", "vm.Name", vm.Name)
				return ctrl.Result{}, err
			}
			// VirtualMachine created successfully - return and requeue
			return ctrl.Result{Requeue: true}, nil
		}

		reqLogger.Error(err, "Failed to get VirtualMachine")
		return ctrl.Result{}, err
	}
	reqLogger.Info("VirtualMachine exists", "vm.Name", vm.Name)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VirtualMachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vmv1alpha1.VirtualMachine{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 2}).
		Complete(r)
}
