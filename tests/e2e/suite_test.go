/*
Copyright © 2022 - 2023 SUSE LLC

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
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rancher-sandbox/ele-testhelpers/kubectl"
	"github.com/rancher-sandbox/ele-testhelpers/rancher"
	"github.com/rancher-sandbox/ele-testhelpers/tools"
	"github.com/rancher/elemental/tests/e2e/helpers/elemental"
)

const (
	appYaml               = "../assets/hello-world_app.yaml"
	backupYaml            = "../assets/backup.yaml"
	clusterYaml           = "../assets/cluster.yaml"
	ciTokenYaml           = "../assets/local-kubeconfig-token-skel.yaml"
	configPrivateCAScript = "../scripts/config-private-ca"
	dumbRegistrationYaml  = "../assets/dumb_machineRegistration.yaml"
	emulateTPMYaml        = "../assets/emulateTPM.yaml"
	getOSScript           = "../scripts/get-name-from-managedosversion"
	httpSrv               = "http://192.168.122.1:8000"
	installConfigYaml     = "../../install-config.yaml"
	installHardenedScript = "../scripts/config-hardened"
	installVMScript       = "../scripts/install-vm"
	localKubeconfigYaml   = "../assets/local-kubeconfig-skel.yaml"
	netDefaultFileName    = "../assets/net-default.xml"
	numberOfNodesMax      = 30
	registrationYaml      = "../assets/machineRegistration.yaml"
	restoreYaml           = "../assets/restore.yaml"
	seedimageYaml         = "../assets/seedImage.yaml"
	selectorYaml          = "../assets/selector.yaml"
	upgradeSkelYaml       = "../assets/upgrade_skel.yaml"
	userName              = "root"
	userPassword          = "r0s@pwd1"
	vmNameRoot            = "node"
)

var (
	arch                  string
	backupRestoreVersion  string
	caType                string
	CertManagerVersion    string
	clusterName           string
	clusterNS             string
	clusterType           string
	elementalSupport      string
	emulateTPM            bool
	rancherHostname       string
	isoBoot               string
	k8sUpstreamVersion    string
	k8sVersion            string
	numberOfVMs           int
	operatorUpgrade       string
	operatorRepo          string
	poolType              string
	proxy                 string
	rancherChannel        string
	rancherLogCollector   string
	rancherVersion        string
	rancherUpgrade        string
	rancherUpgradeChannel string
	rancherUpgradeVersion string
	sequential            bool
	testType              string
	upgradeImage          string
	upgradeOSChannel      string
	upgradeType           string
	usedNodes             int
	vmIndex               int
	vmName                string
)

func CheckClusterState(ns, cluster string) {
	type state struct {
		conditionStatus string
		conditionType   string
	}

	// List of conditions to check
	states := []state{
		{
			conditionStatus: "True",
			conditionType:   "AgentDeployed",
		},
		{
			conditionStatus: "True",
			conditionType:   "NoDiskPressure",
		},
		{
			conditionStatus: "True",
			conditionType:   "NoMemoryPressure",
		},
		{
			conditionStatus: "True",
			conditionType:   "Provisioned",
		},
		{
			conditionStatus: "True",
			conditionType:   "Ready",
		},
		{
			conditionStatus: "False",
			conditionType:   "Reconciling",
		},
		{
			conditionStatus: "False",
			conditionType:   "Stalled",
		},
		{
			conditionStatus: "True",
			conditionType:   "Updated",
		},
		{
			conditionStatus: "True",
			conditionType:   "Waiting",
		},
	}

	// Check that the cluster is in Ready state (this means that it has been created)
	Eventually(func() string {
		status, _ := kubectl.Run("get", "cluster",
			"--namespace", ns, cluster,
			"-o", "jsonpath={.status.ready}")
		return status
	}, tools.SetTimeout(2*time.Duration(usedNodes)*time.Minute), 10*time.Second).Should(Equal("true"))

	// Check that all needed conditions are in the good state
	for _, s := range states {
		Eventually(func() string {
			status, _ := kubectl.Run("get", "cluster",
				"--namespace", ns, cluster,
				"-o", "jsonpath={.status.conditions[?(@.type==\""+s.conditionType+"\")].status}")

			// Show the status in case of issue, easier to debug
			if status != s.conditionStatus {
				GinkgoWriter.Printf("!! Cluster status issue !!  %s is %s instead of %s\n",
					s.conditionType, status, s.conditionStatus)
			}

			return status
		}, tools.SetTimeout(2*time.Duration(usedNodes)*time.Minute), 10*time.Second).Should(Equal(s.conditionStatus))
	}
}

func GetNodeInfo(hostName string) (*tools.Client, string) {
	// Get network data
	hostData, err := rancher.GetHostNetConfig(".*name=\""+hostName+"\".*", netDefaultFileName)
	Expect(err).To(Not(HaveOccurred()))

	// Set 'client' to be able to access the node through SSH
	c := &tools.Client{
		Host:     string(hostData.IP) + ":22",
		Username: userName,
		Password: userPassword,
	}

	return c, hostData.Mac
}

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
	backupRestoreVersion = os.Getenv("BACKUP_RESTORE_VERSION")
	caType = os.Getenv("CA_TYPE")
	CertManagerVersion = os.Getenv("CERT_MANAGER_VERSION")
	clusterName = os.Getenv("CLUSTER_NAME")
	clusterNS = os.Getenv("CLUSTER_NS")
	clusterType = os.Getenv("CLUSTER_TYPE")
	elementalSupport = os.Getenv("ELEMENTAL_SUPPORT")
	eTPM := os.Getenv("EMULATE_TPM")
	rancherHostname = os.Getenv("PUBLIC_DNS")
	index := os.Getenv("VM_INDEX")
	isoBoot = os.Getenv("ISO_BOOT")
	k8sUpstreamVersion = os.Getenv("K8S_UPSTREAM_VERSION")
	k8sVersion = os.Getenv("K8S_VERSION_TO_PROVISION")
	number := os.Getenv("VM_NUMBERS")
	operatorUpgrade = os.Getenv("OPERATOR_UPGRADE")
	operatorRepo = os.Getenv("OPERATOR_REPO")
	poolType = os.Getenv("POOL")
	proxy = os.Getenv("PROXY")
	rancherLogCollector = os.Getenv("RANCHER_LOG_COLLECTOR")
	rancherVersion = os.Getenv("RANCHER_VERSION")
	rancherUpgrade = os.Getenv("RANCHER_UPGRADE")
	seqString := os.Getenv("SEQUENTIAL")
	testType = os.Getenv("TEST_TYPE")
	upgradeImage = os.Getenv("UPGRADE_IMAGE")
	upgradeOSChannel = os.Getenv("UPGRADE_OS_CHANNEL")
	upgradeType = os.Getenv("UPGRADE_TYPE")

	// Only if VM_INDEX is set
	if index != "" {
		var err error
		vmIndex, err = strconv.Atoi(index)
		Expect(err).To(Not(HaveOccurred()))

		// Set default hostname
		vmName = elemental.SetHostname(vmNameRoot, vmIndex)
	} else {
		// Default value for vmIndex
		vmIndex = 0
	}

	// Only if VM_NUMBER is set
	if number != "" {
		var err error
		numberOfVMs, err = strconv.Atoi(number)
		Expect(err).To(Not(HaveOccurred()))
	} else {
		// By default set to vmIndex
		numberOfVMs = vmIndex
	}

	// Set number of "used" nodes
	// NOTE: could be the number added nodes or the number of nodes to use/upgrade
	usedNodes = (numberOfVMs - vmIndex) + 1

	// Force a correct value for emulateTPM
	switch eTPM {
	case "true":
		emulateTPM = true
	default:
		emulateTPM = false
	}

	// Same for sequential
	switch seqString {
	case "true":
		sequential = true
	default:
		sequential = false
	}

	// Extract Rancher Manager channel/version to install
	if rancherVersion != "" {
		s := strings.Split(rancherVersion, "/")
		rancherChannel = s[0]
		rancherVersion = s[1]
	}

	// Extract Rancher Manager channel/version to upgrade
	if rancherUpgrade != "" {
		s := strings.Split(rancherUpgrade, "/")
		rancherUpgradeChannel = s[0]
		rancherUpgradeVersion = s[1]
	}

	// Start HTTP server
	tools.HTTPShare("../..", ":8000")
})
