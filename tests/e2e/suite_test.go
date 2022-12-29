/*
Copyright © 2022 SUSE LLC

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

package e2e_test

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rancher/elemental/tests/e2e/helpers/misc"
)

const (
	clusterYaml               = "../assets/cluster.yaml"
	emulatedTPMYaml           = "../assets/emulated_tpm.yaml"
	configPrivateCAScript     = "../scripts/config-private-ca"
	installConfigYaml         = "../../install-config.yaml"
	installVMScript           = "../scripts/install-vm"
	netDefaultFileName        = "../assets/net-default.xml"
	osListYaml                = "../assets/managedOSVersionChannel.yaml"
	registrationYaml          = "../assets/machineRegistration.yaml"
	selectorYaml              = "../assets/selector.yaml"
	upgradeClusterTargetsYaml = "../assets/upgrade_clusterTargets.yaml"
	upgradeOSVersionNameYaml  = "../assets/upgrade_managedOSVersionName.yaml"
	userName                  = "root"
	userPassword              = "r0s@pwd1"
	vmNameRoot                = "node"
)

var (
	arch                string
	caType              string
	clusterName         string
	clusterNS           string
	elementalSupport    string
	emulateTPM          string
	imageVersion        string
	isoBoot             string
	k8sVersion          string
	osImage             string
	rancherChannel      string
	rancherLogCollector string
	rancherVersion      string
	testType            string
	upgradeType         string
	upgradeOperator     string
	vmIndex             int
	vmName              string
)

func FailWithReport(message string, callerSkip ...int) {
	// Ensures the correct line numbers are reported
	Fail(message, callerSkip[0]+1)
}

func TestE2E(t *testing.T) {
	RegisterFailHandler(FailWithReport)
	RunSpecs(t, "Elemental End-To-End Test Suite")
}

var _ = BeforeSuite(func() {
	arch = os.Getenv("ARCH")
	caType = os.Getenv("CA_TYPE")
	clusterName = os.Getenv("CLUSTER_NAME")
	clusterNS = os.Getenv("CLUSTER_NS")
	elementalSupport = os.Getenv("ELEMENTAL_SUPPORT")
	emulateTPM = os.Getenv("EMULATE_TPM")
	imageVersion = os.Getenv("IMAGE_VERSION")
	index := os.Getenv("VM_INDEX")
	isoBoot = os.Getenv("ISO_BOOT")
	k8sVersion = os.Getenv("K8S_VERSION_TO_PROVISION")
	osImage = os.Getenv("CONTAINER_IMAGE")
	rancherChannel = os.Getenv("RANCHER_CHANNEL")
	rancherLogCollector = os.Getenv("RANCHER_LOG_COLLECTOR")
	rancherVersion = os.Getenv("RANCHER_VERSION")
	testType = os.Getenv("TEST_TYPE")
	upgradeType = os.Getenv("UPGRADE_TYPE")
	upgradeOperator = os.Getenv("UPGRADE_OPERATOR")

	// Only if VM_INDEX is set
	if index != "" {
		var err error
		vmIndex, err = strconv.Atoi(index)
		Expect(err).To(Not(HaveOccurred()))

		// Now we can set the VM name
		vmName = vmNameRoot + "-" + fmt.Sprintf("%03d", vmIndex)
	}

	// Force a correct value
	if emulateTPM != "true" {
		emulateTPM = "false"
	}

	// Start HTTP server
	misc.FileShare("../..", ":8000")
})
