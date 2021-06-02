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
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	vmv1alpha1 "github.com/tmax-cloud/hypercloud-ovirt-operator/api/v1alpha1"
	"github.com/tmax-cloud/hypercloud-ovirt-operator/pkg/ovirt"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = vmv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())

	const (
		SecretName = "ovirt-master"

		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	vmr := &VirtualMachineReconciler{
		Client: k8sManager.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("VirtualMachine"),
		Scheme: k8sManager.GetScheme(),
	}
	vmr.Register(ovirt.NewListener())
	err = (vmr).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	secret := &corev1.Secret{
		TypeMeta: v1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      SecretName,
			Namespace: corev1.NamespaceDefault,
		},
		Type: "Opaque",
		StringData: map[string]string{
			"url":  "https://node1.test.dom/ovirt-engine/api",
			"name": "admin@internal",
			"pass": "1",
		},
	}
	Expect(k8sClient.Create(context.Background(), secret)).Should(Succeed())

	secretLookupKey := types.NamespacedName{Name: SecretName, Namespace: corev1.NamespaceDefault}
	createdSecret := &corev1.Secret{}
	Eventually(func() error {
		return k8sClient.Get(context.Background(), secretLookupKey, createdSecret)
	}, timeout, interval).Should(Succeed())

	Expect(string(createdSecret.Data["url"])).Should(Equal("https://node1.test.dom/ovirt-engine/api"))
	Expect(string(createdSecret.Data["name"])).Should(Equal("admin@internal"))
	Expect(string(createdSecret.Data["pass"])).Should(Equal("1"))

	go func() {
		err = k8sManager.Start(ctrl.SetupSignalHandler())
		Expect(err).ToNot(HaveOccurred())
	}()
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
