package ovirt

import (
	"testing"
	"time"

	"github.com/tmax-cloud/hypercloud-ovirt-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getActuator() *OvirtListener {
	a := NewListener()
	secret := &v1.Secret{Data: map[string][]byte{
		"url":  []byte("https://node1.test.dom/ovirt-engine/api"),
		"name": []byte("admin@internal"),
		"pass": []byte("1"),
	}}
	a.SetListener(secret)
	return a
}

func getVM() *v1alpha1.VirtualMachine {
	vm := &v1alpha1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{Name: "actuator-test"},
		Spec:       v1alpha1.VirtualMachineSpec{Template: "Blank"},
	}
	return vm
}

func TestSetListener(t *testing.T) {
	expected := &OvirtListener{url: "", name: "", pass: ""}
	actual := NewListener()
	secret := &v1.Secret{}
	actual.SetListener(secret)

	if expected.url != actual.url || expected.name != actual.name || expected.pass != actual.pass {
		t.Error("expected:", expected, "actual:", actual)
	}

	expected = &OvirtListener{
		url:  "https://node1.test.dom/ovirt-engine/api",
		name: "admin@internal",
		pass: "1"}
	actual = getActuator()

	if expected.url != actual.url || expected.name != actual.name || expected.pass != actual.pass {
		t.Error("expected:", expected, "actual:", actual)
	}
}

func TestGetVM(t *testing.T) {
	a := getActuator()
	vm := &v1alpha1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{Name: "non-existing-name"},
		Spec:       v1alpha1.VirtualMachineSpec{Template: "Blank"},
	}
	err := a.GetVM(vm)
	if err != nil {
		if errors.IsNotFound(err) {
			t.Log(err)
			return
		}
		t.Error(err)
	}
}

func TestAddVM(t *testing.T) {
	a := getActuator()
	vm := getVM()
	err := a.AddVM(vm)
	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(Timeout)
	err = a.GetVM(vm)
	if err != nil {
		t.Error(err)
	}
}

func TestFinalizeVm(t *testing.T) {
	a := getActuator()
	err := a.FinalizeVm(getVM())
	if err != nil {
		t.Error(err)
	}
}
