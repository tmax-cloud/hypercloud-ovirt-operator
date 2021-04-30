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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	vmv1alpha1 "github.com/tmax-cloud/hypercloud-ovirt-operator/api/v1alpha1"
	vmtypes "github.com/tmax-cloud/hypercloud-ovirt-operator/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("VirtualMachine controller", func() {
	const (
		VirtualmachineName = "test-virtualmachine"

		timeout  = time.Second * 10
		duration = time.Second * 30
		interval = time.Millisecond * 250
	)

	Context("When creating a new VirtualMachine", func() {
		It("Should update VirtualMachine Status.Conditions[Ready] to true when new VirtualMachine are created", func() {
			ctx := context.Background()
			vm := &vmv1alpha1.VirtualMachine{
				TypeMeta: v1.TypeMeta{
					APIVersion: "vm.tmaxcloud.com/v1alpha1",
					Kind:       "VirtualMachine",
				},
				ObjectMeta: v1.ObjectMeta{
					Name:      VirtualmachineName,
					Namespace: corev1.NamespaceDefault,
				},
				Spec: vmv1alpha1.VirtualMachineSpec{
					Template: "ubuntu-18.04-cloudimg",
				},
			}
			Expect(k8sClient.Create(ctx, vm)).Should(Succeed())

			vmLookupKey := types.NamespacedName{Name: VirtualmachineName, Namespace: corev1.NamespaceDefault}
			createdVm := &vmv1alpha1.VirtualMachine{}
			Eventually(func() error {
				return k8sClient.Get(ctx, vmLookupKey, createdVm)
			}, timeout, interval).Should(Succeed())
			Expect(createdVm.Spec.Template).Should(Equal("ubuntu-18.04-cloudimg"))

			By("By adding the VM Ready condition")
			meta.SetStatusCondition(&createdVm.Status.Conditions, v1.Condition{
				Type:    vmtypes.VmReady,
				Status:  v1.ConditionTrue,
				Reason:  "VmReady",
				Message: "VM is now ready",
			})

			Eventually(func() (v1.ConditionStatus, error) {
				conditions := createdVm.Status.Conditions
				if conditions == nil {
					return "", nil
				}
				condition := meta.FindStatusCondition(createdVm.Status.Conditions, vmtypes.VmReady)
				return condition.Status, nil
			}, duration, interval).Should(Equal(v1.ConditionTrue))
		})
	})
})
