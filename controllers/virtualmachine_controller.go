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
	"time"

	"github.com/go-logr/logr"
	ovirtsdk4 "github.com/ovirt/go-ovirt"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	vmv1alpha1 "github.com/tmax-cloud/hypercloud-ovirt-operator/api/v1alpha1"
)

// VirtualMachineReconciler reconciles a VirtualMachine object
type VirtualMachineReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
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

	// Check if the VirtualMachine already exists, if not create a new one
	err = r.getVM(vm.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Creating a new VirtualMachine", "vm.Name", vm.Name)
			// VirtualMachine created successfully - return and requeue
			return ctrl.Result{Requeue: true}, nil
		}

		log.Error(err, "Failed to get VirtualMachine")
		return ctrl.Result{}, err
	}
	log.Info("VirtualMachine exists", "vm.Name", vm.Name)

	return ctrl.Result{}, nil
}

func (r *VirtualMachineReconciler) getVM(name string) error {
	log := r.Log.WithName("virtualmachine").WithName("ovirt sdk")
	inputRawURL := "https://node1.test.dom/ovirt-engine/api" // TODO: remove the URL link

	conn, err := ovirtsdk4.NewConnectionBuilder().
		URL(inputRawURL).
		Username("admin@internal").
		Password("1"). // TODO: secure the password
		Insecure(true).
		Compress(true).
		Timeout(time.Second * 10).
		Build()
	if err != nil {
		log.Error(err, "Make connection failed")
		return err
	}
	defer conn.Close()

	vmsService := conn.SystemService().VmsService()
	vmsResponse, err := vmsService.List().Send()

	if err != nil {
		log.Error(err, "Failed to get vm list")
		return err
	}
	if vms, ok := vmsResponse.Vms(); ok {
		for _, vm := range vms.Slice() {
			if vmName, ok := vm.Name(); ok && vmName == name {
				return nil
			}
		}
	}

	return errors.NewNotFound(schema.GroupResource{}, name)
}

// SetupWithManager sets up the controller with the Manager.
func (r *VirtualMachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vmv1alpha1.VirtualMachine{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 2}).
		Complete(r)
}
