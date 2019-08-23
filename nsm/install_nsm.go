// Copyright 2019 The Meshery Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package nsm

import (
	// "io"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"time"

	git "gopkg.in/src-d/go-git.v4" // Apache v2.0

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	repoURL     = "https://github.com/networkservicemesh/networkservicemesh.git"
	cachePeriod = 24 * time.Hour
)

var (
	destinationFolder             = path.Join(os.TempDir(), "NetworkServiceMesh")
	clusterroleadminfile          = path.Join(destinationFolder, "k8s/conf/cluster-role-admin.yaml")
	clusterrolebindingfile        = path.Join(destinationFolder, "k8s/conf/cluster-role-binding.yaml")
	clusterroleviewfile           = path.Join(destinationFolder, "k8s/conf/cluster-role-view.yaml")
	crdnetworkserviceendpointfile = path.Join(destinationFolder, "k8s/conf/crd-networkserviceendpoints.yaml")
	crdnetworkservicefile         = path.Join(destinationFolder, "k8s/conf/crd-networkservices.yaml")
	crdnetworkservicemanagerfile  = path.Join(destinationFolder, "k8s/conf/crd-networkservicemanagers.yaml")
	nsmgrfile                     = path.Join(destinationFolder, "k8s/conf/nsmgr.yaml")
	vppagentdataplanefile         = path.Join(destinationFolder, "k8s/conf/vppagent-dataplane.yaml")
	crossconnectmonitorfile       = path.Join(destinationFolder, "k8s/conf/crossconnect-monitor.yaml")
	admissionwebhookfile          = path.Join(destinationFolder, "k8s/conf/admission-webhook.yaml")
	skydivefile                   = path.Join(destinationFolder, "k8s/conf/skydive.yaml")
	webhookcertificatefile        = path.Join(destinationFolder, "scripts/webhook-create-cert.sh")
	icmpnscfile                   = path.Join(destinationFolder, "k8s/conf/nsc.yaml")
	icmprespondernsefile          = path.Join(destinationFolder, "k8s/conf/icmp-responder-nse.yaml")
	icmpvppagentnscfile           = path.Join(destinationFolder, "k8s/conf/vppagent-nsc.yaml")
	icmpvppagentrespondernsefile  = path.Join(destinationFolder, "k8s/conf/vppagent-icmp-responder-nse.yaml")
)

func (nsmClient *NSMClient) downloadNSM() {
	fmt.Println("Attempting to download the NSM project...")
	//        Info("git clone %s %s --recursive", "https://github.com/networkservicemesh/networkservicemesh.git","/home/harshini/Documents/networkservicemesh")

	// Create temp directory to clone the repository within.
	_, err := os.Stat(destinationFolder)
	if err == nil {
		fmt.Println("The repo already exists ")
	}
	if os.IsNotExist(err) {
		fmt.Println("Creating temporary directory for repo clone...")
		err := os.MkdirAll(destinationFolder, os.ModePerm)
		if err != nil {
			err = errors.Wrapf(err, "unable to create a folder  %s", destinationFolder)
			logrus.Error(err)

		}

		// CLean up temporary directory when done.
		//	defer os.RemoveAll(dir)

		// Clone the repository into the temp dir
		fmt.Println("Cloning NSM repo...")
		_, err = git.PlainClone(destinationFolder, false, &git.CloneOptions{
			URL: repoURL,
		})

		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Clone of NSM repo completed in ", destinationFolder)
	}

}

func (nsmClient *NSMClient) getComponentYAML(fileName string) (string, error) {

	/*specificVersionName, err := iClient.downloadIstio()
	if err != nil {
		return "", err
	}
	installFileLoc := fmt.Sprintf(fileName, specificVersionName)
	logrus.Debugf("checking if install file exists at path: %s", installFileLoc)
	_, err = os.Stat(installFileLoc)
	if err != nil {
		if os.IsNotExist(err) {
			logrus.Error(err)
			return "", err
		} else {
			err = errors.Wrap(err, "unknown error")
			logrus.Error(err)
			return "", err
		}
	}*/
	fileContents, err := ioutil.ReadFile(fileName)
	if err != nil {
		err = errors.Wrap(err, "unable to read file")
		logrus.Error(err)
		return "", err
	}
	return string(fileContents), nil
}

func (nsmClient *NSMClient) getClusterRoleadminYAML() (string, error) {
	return nsmClient.getComponentYAML(clusterroleadminfile)
}

func (nsmClient *NSMClient) getClusterRoleViewYaml() (string, error) {
	return nsmClient.getComponentYAML(clusterroleviewfile)
}
func (nsmClient *NSMClient) getClusterBindingYaml() (string, error) {
	return nsmClient.getComponentYAML(clusterrolebindingfile)
}
func (nsmClient *NSMClient) getCrdNetworkserviceEndpointYaml() (string, error) {
	return nsmClient.getComponentYAML(crdnetworkserviceendpointfile)
}
func (nsmClient *NSMClient) getCrdNetworkserviceYaml() (string, error) {
	return nsmClient.getComponentYAML(crdnetworkservicefile)
}
func (nsmClient *NSMClient) getCrdNetworkserviceManagerYaml() (string, error) {
	return nsmClient.getComponentYAML(crdnetworkservicemanagerfile)
}
func (nsmClient *NSMClient) getNsmgrYaml() (string, error) {
	return nsmClient.getComponentYAML(nsmgrfile)
}
func (nsmClient *NSMClient) getVppagentdataplaneYaml() (string, error) {
	return nsmClient.getComponentYAML(vppagentdataplanefile)
}
func (nsmClient *NSMClient) getCrossconnectYaml() (string, error) {
	return nsmClient.getComponentYAML(crossconnectmonitorfile)

}
func (nsmClient *NSMClient) getAdmissionWebhookYaml() (string, error) {
	return nsmClient.getComponentYAML(admissionwebhookfile)

}
func (nsmClient *NSMClient) getSkydiveYaml() (string, error) {
	return nsmClient.getComponentYAML(skydivefile)

}
func (nsmClient *NSMClient) getICMPAppYaml() (string, error) {
	buf := bytes.Buffer{}
	var fileContents string
	fileContents, _ = nsmClient.getICMPNSCYaml()
	buf.WriteString(fileContents)
	fileContents, _ = nsmClient.getICMPNSEYaml()
	buf.WriteString(fileContents)
	fileContents, _ = nsmClient.getICMPVppagentNSCYaml()
	buf.WriteString(fileContents)
	fileContents, _ = nsmClient.getICMPVppagentNSEYaml()
	buf.WriteString(fileContents)
	return buf.String(), nil
}
func (nsmClient *NSMClient) getICMPNSCYaml() (string, error) {
	return nsmClient.getComponentYAML(icmpnscfile)

}
func (nsmClient *NSMClient) getICMPNSEYaml() (string, error) {
	return nsmClient.getComponentYAML(icmprespondernsefile)

}
func (nsmClient *NSMClient) getICMPVppagentNSCYaml() (string, error) {
	return nsmClient.getComponentYAML(icmpvppagentnscfile)

}
func (nsmClient *NSMClient) getICMPVppagentNSEYaml() (string, error) {
	return nsmClient.getComponentYAML(icmpvppagentrespondernsefile)

}
