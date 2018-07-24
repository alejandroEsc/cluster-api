/*
Copyright 2018 The Kubernetes Authors.
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

package maas

import(
	"github.com/golang/glog"

	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/cluster-api/pkg/cert"
	"sigs.k8s.io/cluster-api/pkg/kubeadm"
	client "sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset/typed/cluster/v1alpha1"

	"time"
	"strings"
	"sigs.k8s.io/cluster-api/cloud/google/machinesetup"
)

type MachineActuator struct {
	CertificateAuthority     *cert.CertificateAuthority
	kubeadm 				 *kubeadm.Kubeadm
}

type MachineActuatorParams struct {
	V1Alpha1Client           client.ClusterV1alpha1Interface
	MachineSetupConfigGetter *machinesetup.ConfigWatch
}

func (ma *MachineActuator) getKubeadmToken() (string, error) {
	tokenParams := kubeadm.TokenCreateParams{
		Ttl: time.Duration(10) * time.Minute,
	}

	output, err := ma.kubeadm.TokenCreate(tokenParams)
	if err != nil {
		glog.Errorf("unable to create token: %v", err)
		return "", err
	}

	return strings.TrimSpace(output), err
}


// NewMachineActuator creates a new actuator to manage manchines.
func NewMachineActuator(params MachineActuatorParams) (*MachineActuator, error) {
	return &MachineActuator{
	}, nil
}


// GetIP returns the IP where a master node is created.
func (ma *MachineActuator) GetIP(cluster *clusterv1.Cluster, machine *clusterv1.Machine) (string, error) {
	return "", nil
}

// GetKubeConfig returns the kubeconfig file.
func  (ma *MachineActuator) GetKubeConfig(cluster *clusterv1.Cluster, master *clusterv1.Machine) (string, error) {
	return "", nil
}

// Create the machine.
func Create(*clusterv1.Cluster, *clusterv1.Machine) error {
	return nil
}
// Delete the machine.
func Delete(*clusterv1.Cluster, *clusterv1.Machine) error {
	return nil
}
// Update the machine to the provided definition.
func Update(*clusterv1.Cluster, *clusterv1.Machine) error {
	return nil
}
// Checks if the machine currently exists.
func Exists(*clusterv1.Cluster, *clusterv1.Machine) (bool, error) {
	return false, nil
}


