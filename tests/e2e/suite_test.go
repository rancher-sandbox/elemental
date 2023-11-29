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
	"regexp"
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
	ciTokenYaml           = "../assets/local-kubeconfig-token-skel.yaml"
	configPrivateCAScript = "../scripts/config-private-ca"
	configRKE2Yaml        = "../assets/config_rke2.yaml"
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
	resetMachineInv       = "../assets/reset_machine_inventory.yaml"
	restoreYaml           = "../assets/restore.yaml"
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
	clusterYaml           string
	elementalSupport      string
	emulateTPM            bool
	rancherHostname       string
	isoBoot               bool
	k8sUpstreamVersion    string
	k8sVersion            string
	numberOfClusters      int
	numberOfVMs           int
	operatorUpgrade       string
	operatorRepo          string
	os2Test               string
	poolType              string
	proxy                 string
	rancherChannel        string
	rancherHeadVersion    string
	rancherLogCollector   string
	rancherVersion        string
	rancherUpgrade        string
	rancherUpgradeChannel string
	rancherUpgradeVersion string
	registrationYaml      string
	seedImageYaml         string
	selectorYaml          string
	sequential            bool
	testType              string
	upgradeImage          string
	upgradeOSChannel      string
	upgradeType           string
	usedNodes             int
	vmIndex               int
	vmName                string
)

/**
 * Wait for cluster to be in a stable state
 * @param ns Namespace where the cluster is deployed
 * @param cn Cluster resource name
 * @returns Nothing, the function will fail through Ginkgo in case of issue
 */
func WaitCluster(ns, cn string) {
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
			"--namespace", ns, cn,
			"-o", "jsonpath={.status.ready}")
		return status
	}, tools.SetTimeout(2*time.Duration(usedNodes)*time.Minute), 10*time.Second).Should(Equal("true"))

	// Check that all needed conditions are in the good state
	for _, s := range states {
		counter := 0

		Eventually(func() string {
			status, _ := kubectl.Run("get", "cluster",
				"--namespace", ns, cn,
				"-o", "jsonpath={.status.conditions[?(@.type==\""+s.conditionType+"\")].status}")

			if status != s.conditionStatus {
				// Show the status in case of issue, easier to debug (but log after 10 different issues)
				// NOTE: it's not perfect but it's mainly a way to inform that the cluster took time to came up
				counter++
				if counter > 10 {
					GinkgoWriter.Printf("!! Cluster status issue !! %s is %s instead of %s\n",
						s.conditionType, status, s.conditionStatus)

					// Reset counter
					counter = 0
				}

				// Check if rancher-system-agent.service has some issue
				if s.conditionType == "Provisioned" || s.conditionType == "Ready" || s.conditionStatus == "Updated" {
					msg, _ := kubectl.Run("get", "cluster",
						"--namespace", ns, cn,
						"-o", "jsonpath={.status.conditions[?(@.type==\""+s.conditionType+"\")].message}")

					// We can try to restart the rancher-system-agent service on the failing node
					// because sometimes it can fail just because of a sporadic/timeout issue and a restart can fix it!
					if strings.Contains(msg, "check rancher-system-agent.service logs on node") {
						// Get the Elemental hostname and then the *real* hostname
						substr := regexp.MustCompile(`(` + cn + `-.*): error`).FindStringSubmatch(msg)
						for _, node := range strings.Split(substr[1], ",") {
							// Get node IP
							ip, _ := elemental.GetExternalMachineIP(ns, node)

							if ip != "" {
								// Set 'client' to be able to access the node through SSH
								cl := &tools.Client{
									Host:     ip + ":22",
									Username: userName,
									Password: userPassword,
								}

								// Log the workaround, could be useful
								GinkgoWriter.Printf("!! rancher-system-agent issue !! Service has been restarted on %s\n", node)

								// Restart rancher-system-agent service on the node
								// NOTE: wait a little to be sure that all is restarted before continuing
								cl.RunSSH("systemctl restart rancher-system-agent.service")
								time.Sleep(tools.SetTimeout(15 * time.Second))
							}
						}
					}
				}
			}

			return status
		}, tools.SetTimeout(2*time.Duration(usedNodes)*time.Minute), 10*time.Second).Should(Equal(s.conditionStatus))
	}
}

/**
 * Check that Cluster resource has been correctly created
 * @param ns Namespace where the cluster is deployed
 * @param cn Cluster resource name
 * @returns Nothing, the function will fail through Ginkgo in case of issue
 */
func CheckCreatedCluster(ns, cn string) {
	// Check that the cluster is correctly created
	Eventually(func() string {
		out, _ := kubectl.Run("get", "cluster",
			"--namespace", ns,
			cn, "-o", "jsonpath={.metadata.name}")
		return out
	}, tools.SetTimeout(3*time.Minute), 5*time.Second).Should(Equal(cn))
}

/**
 * Check that Cluster resource has been correctly created
 * @param ns Namespace where the cluster is deployed
 * @param rn MachineRegistration resource name
 * @returns Nothing, the function will fail through Ginkgo in case of issue
 */
func CheckCreatedRegistration(ns, rn string) {
	Eventually(func() string {
		out, _ := kubectl.Run("get", "MachineRegistration",
			"--namespace", clusterNS,
			"-o", "jsonpath={.items[*].metadata.name}")
		return out
	}, tools.SetTimeout(3*time.Minute), 5*time.Second).Should(ContainSubstring(rn))
}

/**
 * Check that a SelectorTemplate resource has been correctly created
 * @param ns Namespace where the cluster is deployed
 * @param sn Selector name
 * @returns Nothing, the function will fail through Ginkgo in case of issue
 */
func CheckCreatedSelectorTemplate(ns, sn string) {
	Eventually(func() string {
		out, _ := kubectl.Run("get", "MachineInventorySelectorTemplate",
			"--namespace", ns,
			"-o", "jsonpath={.items[*].metadata.name}")
		return out
	}, tools.SetTimeout(3*time.Minute), 5*time.Second).Should(ContainSubstring(sn))
}

/**
 * Wait for OSVersion to be populated
 * @param ns Namespace where the cluster is deployed
 * @returns Nothing, the function will fail through Ginkgo in case of issue
 */
func WaitForOSVersion(ns string) {
	Eventually(func() string {
		out, _ := kubectl.Run("get", "ManagedOSVersion",
			"--namespace", ns,
			"-o", "jsonpath={.items[*].metadata.name}")
		return out
	}, tools.SetTimeout(2*time.Minute), 5*time.Second).Should(Not(BeEmpty()))
}

/**
 * Check SSH connection
 * @param cl Client (node) informations
 * @returns Nothing, the function will fail through Ginkgo in case of issue
 */
func CheckSSH(cl *tools.Client) {
	Eventually(func() string {
		out, _ := cl.RunSSH("echo SSH_OK")
		return strings.Trim(out, "\n")
	}, tools.SetTimeout(10*time.Minute), 5*time.Second).Should(Equal("SSH_OK"))
}

/**
 * Download ISO built with SeedImage
 * @param ns Namespace where the cluster is deployed
 * @param seedName Name of the used SeedImage resource
 * @param filename Path and name of the file where to store the ISO
 * @returns Nothing, the function will fail through Ginkgo in case of issue
 */
func DownloadBuiltISO(ns, seedName, filename string) {
	// Set minimal ISO file to 500MB
	const minimalISOSize = 500 * 1024 * 1024

	// Check that the seed image is correctly created
	Eventually(func() string {
		out, _ := kubectl.Run("get", "SeedImage",
			"--namespace", ns,
			seedName,
			"-o", "jsonpath={.status}")
		return out
	}, tools.SetTimeout(3*time.Minute), 5*time.Second).Should(ContainSubstring("downloadURL"))

	// Get URL
	seedImageURL, err := kubectl.Run("get", "SeedImage",
		"--namespace", ns,
		seedName,
		"-o", "jsonpath={.status.downloadURL}")
	Expect(err).To(Not(HaveOccurred()))

	// ISO file size should be greater than 500MB
	Eventually(func() int64 {
		// No need to check download status, file size at the end is enough
		_ = tools.GetFileFromURL(seedImageURL, filename, false)
		file, _ := os.Stat(filename)
		return file.Size()
	}, tools.SetTimeout(2*time.Minute), 10*time.Second).Should(BeNumerically(">", minimalISOSize))
}

/**
 * Get Elemental node information
 * @param hn Node hostname
 * @returns Client structure and MAC address
 */
func GetNodeInfo(hn string) (*tools.Client, string) {
	// Get network data
	data, err := rancher.GetHostNetConfig(".*name=\""+hn+"\".*", netDefaultFileName)
	Expect(err).To(Not(HaveOccurred()))

	// Set 'client' to be able to access the node through SSH
	c := &tools.Client{
		Host:     string(data.IP) + ":22",
		Username: userName,
		Password: userPassword,
	}

	return c, data.Mac
}

/**
 * Get Elemental node IP address
 * @param hn Node hostname
 * @returns IP address
 */
func GetNodeIP(hn string) string {
	// Get network data
	data, err := rancher.GetHostNetConfig(".*name=\""+hn+"\".*", netDefaultFileName)
	Expect(err).To(Not(HaveOccurred()))

	return data.IP
}

/**
 * Execute RunHelmBinaryWithCustomErr within a loop with timeout
 * @param s options to pass to RunHelmBinaryWithCustomErr command
 * @returns Nothing, the function will fail through Ginkgo in case of issue
 */
func RunHelmCmdWithRetry(s ...string) {
	Eventually(func() error {
		return kubectl.RunHelmBinaryWithCustomErr(s...)
	}, tools.SetTimeout(2*time.Minute), 20*time.Second).Should(Not(HaveOccurred()))
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
	isoBootString := os.Getenv("ISO_BOOT")
	k8sUpstreamVersion = os.Getenv("K8S_UPSTREAM_VERSION")
	k8sVersion = os.Getenv("K8S_VERSION_TO_PROVISION")
	number := os.Getenv("VM_NUMBERS")
	clusterNumber := os.Getenv("CLUSTER_NUMBER")
	operatorUpgrade = os.Getenv("OPERATOR_UPGRADE")
	operatorRepo = os.Getenv("OPERATOR_REPO")
	os2Test = os.Getenv("OS_TO_TEST")
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

	// Same for isoBoot
	switch isoBootString {
	case "true":
		isoBoot = true
	default:
		isoBoot = false
	}

	// Extract Rancher Manager channel/version to install
	if rancherVersion != "" {
		s := strings.Split(rancherVersion, "/")
		rancherChannel = s[0]
		rancherVersion = s[1]
		rancherHeadVersion = s[2]
	}

	// Extract Rancher Manager channel/version to upgrade
	if rancherUpgrade != "" {
		s := strings.Split(rancherUpgrade, "/")
		rancherUpgradeChannel = s[0]
		rancherUpgradeVersion = s[1]
	}

	// Enable multi-cluster support if needed
	if testType == "multi_cli" {
		if clusterNumber != "" {
			var err error
			numberOfClusters, err = strconv.Atoi(clusterNumber)
			Expect(err).To(Not(HaveOccurred()))
		}

		clusterYaml = "../assets/cluster-multi.yaml"
		registrationYaml = "../assets/machineRegistration-multi.yaml"
		seedImageYaml = "../assets/seedImage-multi.yaml"
		selectorYaml = "../assets/selector-multi.yaml"
	} else {
		clusterYaml = "../assets/cluster.yaml"
		registrationYaml = "../assets/machineRegistration.yaml"
		seedImageYaml = "../assets/seedImage.yaml"
		selectorYaml = "../assets/selector.yaml"
	}

	// Start HTTP server
	tools.HTTPShare("../..", ":8000")
})
