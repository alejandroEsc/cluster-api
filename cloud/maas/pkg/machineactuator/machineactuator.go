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

package machineactuator

import(
	"github.com/golang/glog"

	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/cluster-api/pkg/cert"
	c "github.com/kubernetes-sigs/cluster-api/cloud/maas/pkg/client"
	"sigs.k8s.io/cluster-api/pkg/kubeadm"
	client "sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset/typed/cluster/v1alpha1"
	"github.com/kubernetes-sigs/cluster-api/cloud/maas/pkg/machinesetup"

	"time"
	"strings"
	"os"
	"io/ioutil"
)

type MAASMachineActuator struct {
	CertificateAuthority     *cert.CertificateAuthority
	MAASClient               c.MAASclient
	kubeadm 				 *kubeadm.Kubeadm
}

type MachineActuatorParams struct {
	V1Alpha1Client           client.ClusterV1alpha1Interface
	MachineSetupConfigGetter *machinesetup.ConfigWatch
}

func (ma *MAASMachineActuator) getKubeadmToken() (string, error) {
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

func NewMachineActuator(params MachineActuatorParams) (*GCEClient, error) {
	computeService, err := getOrNewComputeServiceForMachine(params)
	if err != nil {
		return nil, err
	}

	scheme, err := gceconfigv1.NewScheme()
	if err != nil {
		return nil, err
	}
	codec, err := gceconfigv1.NewCodec()
	if err != nil {
		return nil, err
	}

	// Only applicable if it's running inside machine controller pod.
	var privateKeyPath, user string
	if _, err := os.Stat("/etc/sshkeys/private"); err == nil {
		privateKeyPath = "/etc/sshkeys/private"

		b, err := ioutil.ReadFile("/etc/sshkeys/user")
		if err == nil {
			user = string(b)
		} else {
			return nil, err
		}
	}

	return &GCEClient{
		certificateAuthority:   params.CertificateAuthority,
		computeService:         computeService,
		kubeadm:                getOrNewKubeadm(params),
		scheme:                 scheme,
		gceProviderConfigCodec: codec,
		sshCreds: SshCreds{
			privateKeyPath: privateKeyPath,
			user:           user,
		},
		v1Alpha1Client:           params.V1Alpha1Client,
		machineSetupConfigGetter: params.MachineSetupConfigGetter,
	}, nil
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


