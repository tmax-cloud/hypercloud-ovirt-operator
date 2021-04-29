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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	vmv1alpha1 "github.com/tmax-cloud/hypercloud-ovirt-operator/api/v1alpha1"
	"github.com/tmax-cloud/hypercloud-ovirt-operator/pkg/ovirt"
	vmtypes "github.com/tmax-cloud/hypercloud-ovirt-operator/types"
)

const (
	virtualMachineFinalizer = "vm.tmaxcloud.com/finalizer"
)

var ovirtNamespacedName = types.NamespacedName{Name: "ovirt-master", Namespace: "default"}

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
	log := r.Log.WithValues("virtualmachine", req.NamespacedName)
	log.Info("Reconciling VirtualMachine")

	vm := &vmv1alpha1.VirtualMachine{}
	err := r.Get(ctx, req.NamespacedName, vm)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("VirtualMachine resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get VirtualMachine")
		return ctrl.Result{}, err
	}

	if vm.Status.Conditions == nil {
		meta.SetStatusCondition(&vm.Status.Conditions, v1.Condition{
			Type:    vmtypes.VmReady,
			Status:  v1.ConditionUnknown,
			Reason:  "InitVm",
			Message: "Initializing VM resource",
		})
		if err = r.Status().Update(ctx, vm); err != nil {
			log.Error(err, "Failed to update VirtualMachine status")
			return ctrl.Result{}, err
		}
	}

	secret := &corev1.Secret{}
	err = r.Get(ctx, ovirtNamespacedName, secret)
	if err != nil {
		if errors.IsNotFound(err) {
			meta.SetStatusCondition(&vm.Status.Conditions, v1.Condition{
				Type:    vmtypes.VmReady,
				Status:  v1.ConditionFalse,
				Reason:  "SecretNotFound",
				Message: "Ovirt master secret resource is missing",
			})
			if err = r.Status().Update(ctx, vm); err != nil {
				log.Error(err, "Failed to update VirtualMachine status")
				return ctrl.Result{}, err
			}
			log.Error(err, "Secret not found", "secret", ovirtNamespacedName)
			return ctrl.Result{}, err
		}
		meta.SetStatusCondition(&vm.Status.Conditions, v1.Condition{
			Type:    vmtypes.VmReady,
			Status:  v1.ConditionFalse,
			Reason:  "ClientGetFailed",
			Message: "Kubernetes client Get operation failed",
		})
		if err = r.Status().Update(ctx, vm); err != nil {
			log.Error(err, "Failed to update VirtualMachine status")
			return ctrl.Result{}, err
		}
		log.Error(err, "Failed to get Secret")
		return ctrl.Result{}, err
	}

	// set actuator information
	r.Actuator.SetActuator(secret)

	// Check if the VM instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isVmMarkedToBeDeleted := vm.GetDeletionTimestamp() != nil
	if isVmMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(vm, virtualMachineFinalizer) {
			// Run finalization logic for virtualMachineFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.Actuator.FinalizeVm(vm); err != nil {
				log.Error(err, "Failed to finalize VirtualMachine")
				return ctrl.Result{}, err
			}

			// Remove virtualMachineFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(vm, virtualMachineFinalizer)
			if err := r.Update(ctx, vm); err != nil {
				log.Error(err, "Failed to remove finalizer from VirtualMachine")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer for this CR
	if !controllerutil.ContainsFinalizer(vm, virtualMachineFinalizer) {
		controllerutil.AddFinalizer(vm, virtualMachineFinalizer)
		if err = r.Update(ctx, vm); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Check if the VirtualMachine already exists, if not create a new one
	err = r.Actuator.GetVM(vm)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Creating a new VirtualMachine")
			err = r.Actuator.AddVM(vm)
			if err != nil {
				meta.SetStatusCondition(&vm.Status.Conditions, v1.Condition{
					Type:    vmtypes.VmReady,
					Status:  v1.ConditionFalse,
					Reason:  "VmAddFailed",
					Message: "The VM could not be created due to Ovirt Master error",
				})
				if err = r.Status().Update(ctx, vm); err != nil {
					log.Error(err, "Failed to update VirtualMachine status")
					return ctrl.Result{}, err
				}
				log.Error(err, "Failed to create new VirtualMachine")
				return ctrl.Result{}, err
			}
			meta.SetStatusCondition(&vm.Status.Conditions, v1.Condition{
				Type:    vmtypes.VmReady,
				Status:  v1.ConditionTrue,
				Reason:  "VmReady",
				Message: "VM is now ready",
			})
			if err = r.Status().Update(ctx, vm); err != nil {
				log.Error(err, "Failed to update VirtualMachine status")
				return ctrl.Result{}, err
			}
			// VirtualMachine created successfully - return and requeue
			log.Info("Add vm successfully")
			return ctrl.Result{Requeue: true}, nil
		}

		meta.SetStatusCondition(&vm.Status.Conditions, v1.Condition{
			Type:    vmtypes.VmReady,
			Status:  v1.ConditionFalse,
			Reason:  "OvirtClientGetFailed",
			Message: "Ovirt client Get operation failed",
		})
		if err = r.Status().Update(ctx, vm); err != nil {
			log.Error(err, "Failed to update VirtualMachine status")
			return ctrl.Result{}, err
		}
		log.Error(err, "Failed to get VirtualMachine")
		return ctrl.Result{}, err
	}
	log.Info("VirtualMachine exists")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VirtualMachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vmv1alpha1.VirtualMachine{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 2}).
		Complete(r)
}
