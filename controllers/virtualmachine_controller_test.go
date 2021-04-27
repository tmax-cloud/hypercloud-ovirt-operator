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
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	vmv1alpha1 "github.com/tmax-cloud/hypercloud-ovirt-operator/api/v1alpha1"
)

var _ = Describe("VirtualMachine controller", func() {
	const (
		VirtualmachineName      = "test-virtualmachine"
		VirtualmachineNamespace = "default"

		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When updating VirtualMachine Status", func() {
		It("Should increase VirtualMachine Status.Active count when new VirtualMachine are created", func() {
			By("By creating a new VirtualMachine")
			ctx := context.Background()
			vm := &vmv1alpha1.VirtualMachine{
				TypeMeta: v1.TypeMeta{
					APIVersion: "vm.tmaxcloud.com/v1alpha1",
					Kind:       "VirtualMachine",
				},
				ObjectMeta: v1.ObjectMeta{
					Name:      VirtualmachineName,
					Namespace: VirtualmachineNamespace,
				},
				Spec: vmv1alpha1.VirtualMachineSpec{
					Template: "", // Blank
				},
			}
			CurrentGinkgoTestDescription()

			Expect(k8sClient.Create(ctx, vm)).Should(Succeed())

			vmLookupKey := types.NamespacedName{Name: VirtualmachineName, Namespace: VirtualmachineNamespace}
			createdVm := &vmv1alpha1.VirtualMachine{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, vmLookupKey, createdVm)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())
			Expect(createdVm.Spec.Template).Should(Equal(""))
		})
	})
})
