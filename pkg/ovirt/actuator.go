package ovirt

import (
	"time"

	ovirtsdk4 "github.com/ovirt/go-ovirt"
	vmv1alpha1 "github.com/tmax-cloud/hypercloud-ovirt-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	Timeout = 10 * time.Second
)

// OvirtActuator contains connection data
type OvirtActuator struct {
	conn *ovirtsdk4.Connection
	url  string
	name string
	pass string
}

// NewActuator creates new OvirtActuator
func NewActuator() *OvirtActuator {
	return &OvirtActuator{}
}

func (actuator *OvirtActuator) SetActuator(secret *corev1.Secret) {
	actuator.url = string(secret.Data["url"])
	actuator.name = string(secret.Data["name"])
	actuator.pass = string(secret.Data["pass"])
}

func (actuator *OvirtActuator) getConnection() (*ovirtsdk4.Connection, error) {
	var err error
	if actuator.conn == nil || actuator.conn.Test() != nil {
		actuator.conn, err = actuator.createConnection()
	}

	return actuator.conn, err
}

func (actuator *OvirtActuator) createConnection() (*ovirtsdk4.Connection, error) {
	conn, err := ovirtsdk4.NewConnectionBuilder().
		URL(actuator.url).
		Username(actuator.name).
		Password(actuator.pass).
		Insecure(true).
		Compress(true).
		Timeout(Timeout).
		Build()
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// AddVM gets the virtual machine from Ovirt cluster
func (actuator *OvirtActuator) GetVM(m *vmv1alpha1.VirtualMachine) error {
	conn, err := actuator.getConnection()
	if err != nil {
		return err
	}
	defer conn.Close()

	vmsService := conn.SystemService().VmsService()
	vmsResponse, err := vmsService.List().Search("name=" + m.Name).Send()
	if err != nil {
		return err
	}
	vms, _ := vmsResponse.Vms()
	if vm := vms.Slice(); vm != nil {
		return nil
	}

	return errors.NewNotFound(schema.GroupResource{}, m.Name)
}

// AddVM adds the virtual machine to Ovirt cluster
func (actuator *OvirtActuator) AddVM(m *vmv1alpha1.VirtualMachine) error {
	conn, err := actuator.getConnection()
	if err != nil {
		return err
	}
	defer conn.Close()

	vmsService := conn.SystemService().VmsService()
	cluster, err := ovirtsdk4.NewClusterBuilder().Name("Default").Build()
	if err != nil {
		return err
	}
	if m.Spec.Template == "" {
		m.Spec.Template = "Blank"
	}
	template, err := ovirtsdk4.NewTemplateBuilder().Name(m.Spec.Template).Build()
	if err != nil {
		return err
	}
	vm, err := ovirtsdk4.NewVmBuilder().Name(m.Name).Cluster(cluster).Template(template).Build()
	if err != nil {
		return err
	}
	_, err = vmsService.Add().Vm(vm).Send()
	if err != nil {
		return err
	}

	return nil
}

// FinalizeVm removes the virtual machine from Ovirt cluster
func (actuator *OvirtActuator) FinalizeVm(m *vmv1alpha1.VirtualMachine) error {
	conn, err := actuator.getConnection()
	if err != nil {
		return err
	}
	defer conn.Close()

	vmsService := conn.SystemService().VmsService()
	vmsResponse, err := vmsService.List().Search("name=" + m.Name).Send()
	if err != nil {
		return err
	}
	vms, _ := vmsResponse.Vms()
	vmss := vms.Slice()
	if vmss == nil {
		// VM not found in Ovirt. Ignoring since object must be deleted or not created
		return nil
	}
	id, _ := vmss[0].Id()
	vmService := vmsService.VmService(id)
	_, err = vmService.Remove().Send()
	if err != nil {
		return err
	}

	return nil
}
