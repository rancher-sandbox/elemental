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
	"os"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rancher-sandbox/ele-testhelpers/kubectl"
	"github.com/rancher-sandbox/ele-testhelpers/tools"
	"github.com/rancher/elemental/tests/e2e/helpers/misc"
)

var _ = Describe("E2E - Install Rancher Manager", Label("install"), func() {
	// Create kubectl context
	// Default timeout is too small, so New() cannot be used
	k := &kubectl.Kubectl{
		Namespace:    "",
		PollTimeout:  misc.SetTimeout(300 * time.Second),
		PollInterval: 500 * time.Millisecond,
	}

	It("Install Rancher Manager", func() {
		By("Installing K3s", func() {
			// Get K3s installation script
			fileName := "k3s-install.sh"
			err := tools.GetFileFromURL("https://get.k3s.io", fileName, true)
			Expect(err).To(Not(HaveOccurred()))

			// Execute K3s installation
			out, err := exec.Command("sh", fileName).CombinedOutput()
			GinkgoWriter.Printf("%s\n", out)
			Expect(err).To(Not(HaveOccurred()))
		})

		By("Starting K3s", func() {
			err := exec.Command("sudo", "systemctl", "start", "k3s").Run()
			Expect(err).To(Not(HaveOccurred()))

			// Delay few seconds before checking
			time.Sleep(misc.SetTimeout(20 * time.Second))
		})

		By("Waiting for K3s to be started", func() {
			// Wait for all pods to be started
			err := k.WaitForPod("kube-system", "app=local-path-provisioner", "local-path-provisioner")
			Expect(err).To(Not(HaveOccurred()))

			err = k.WaitForPod("kube-system", "k8s-app=kube-dns", "coredns")
			Expect(err).To(Not(HaveOccurred()))

			err = k.WaitForPod("kube-system", "k8s-app=metrics-server", "metrics-server")
			Expect(err).To(Not(HaveOccurred()))

			err = k.WaitForPod("kube-system", "app.kubernetes.io/name=traefik", "traefik")
			Expect(err).To(Not(HaveOccurred()))

			err = k.WaitForPod("kube-system", "svccontroller.k3s.cattle.io/svcname=traefik", "svclb-traefik")
			Expect(err).To(Not(HaveOccurred()))
		})

		if caType == "private" {
			By("Configuring Private CA", func() {
				cmd := exec.Command("../scripts/config-private-ca")
				out, err := cmd.CombinedOutput()
				GinkgoWriter.Printf("%s\n", out)
				Expect(err).To(Not(HaveOccurred()))
			})
		} else {
			By("Installing CertManager", func() {
				err := kubectl.RunHelmBinaryWithCustomErr("repo", "add", "jetstack", "https://charts.jetstack.io")
				Expect(err).To(Not(HaveOccurred()))

				err = kubectl.RunHelmBinaryWithCustomErr("repo", "update")
				Expect(err).To(Not(HaveOccurred()))

				err = kubectl.RunHelmBinaryWithCustomErr("upgrade", "--install", "cert-manager", "jetstack/cert-manager",
					"--namespace", "cert-manager",
					"--create-namespace",
					"--set", "installCRDs=true",
				)
				Expect(err).To(Not(HaveOccurred()))

				err = k.WaitForNamespaceWithPod("cert-manager", "app.kubernetes.io/component=controller")
				Expect(err).To(Not(HaveOccurred()))

				err = k.WaitForNamespaceWithPod("cert-manager", "app.kubernetes.io/component=webhook")
				Expect(err).To(Not(HaveOccurred()))

				err = k.WaitForNamespaceWithPod("cert-manager", "app.kubernetes.io/component=cainjector")
				Expect(err).To(Not(HaveOccurred()))
			})
		}

		By("Installing Rancher", func() {
			err := kubectl.RunHelmBinaryWithCustomErr("repo", "add", "rancher",
				"https://releases.rancher.com/server-charts/"+rancherChannel,
			)
			Expect(err).To(Not(HaveOccurred()))

			err = kubectl.RunHelmBinaryWithCustomErr("repo", "update")
			Expect(err).To(Not(HaveOccurred()))

			// Set flags for Rancher Manager installation
			hostname := os.Getenv("HOSTNAME")
			flags := []string{
				"upgrade", "--install", "rancher", "rancher/rancher",
				"--namespace", "cattle-system",
				"--create-namespace",
				"--set", "hostname=" + hostname,
				"--set", "extraEnv[0].name=CATTLE_SERVER_URL",
				"--set", "extraEnv[0].value=https://" + hostname,
				"--set", "extraEnv[1].name=CATTLE_BOOTSTRAP_PASSWORD",
				"--set", "extraEnv[1].value=rancherpassword",
				"--set", "replicas=1",
			}

			// Set specified version if needed
			if rancherVersion != "" && rancherVersion != "latest" {
				if rancherVersion == "devel" {
					flags = append(flags, "--devel")
				} else {
					flags = append(flags, "--version", rancherVersion)
				}
			}

			// For Private CA
			if caType == "private" {
				flags = append(flags,
					"--set", "ingress.tls.source=secret",
					"--set", "privateCA=true",
				)
			}

			err = kubectl.RunHelmBinaryWithCustomErr(flags...)
			Expect(err).To(Not(HaveOccurred()))

			// Inject secret for Private CA
			if caType == "private" {
				_, err := kubectl.Run("create", "secret",
					"--namespace", "cattle-system",
					"tls", "tls-rancher-ingress",
					"--cert=tls.crt",
					"--key=tls.key",
				)
				Expect(err).To(Not(HaveOccurred()))

				_, err = kubectl.Run("create", "secret",
					"--namespace", "cattle-system",
					"generic", "tls-ca",
					"--from-file=cacerts.pem=./cacerts.pem",
				)
				Expect(err).To(Not(HaveOccurred()))
			}

			err = k.WaitForNamespaceWithPod("cattle-system", "app=rancher")
			Expect(err).To(Not(HaveOccurred()))

			err = k.WaitForNamespaceWithPod("cattle-fleet-local-system", "app=fleet-agent")
			Expect(err).To(Not(HaveOccurred()))

			err = k.WaitForNamespaceWithPod("cattle-system", "app=rancher-webhook")
			Expect(err).To(Not(HaveOccurred()))

			// Check issuer for Private CA
			if caType == "private" {
				out, err := exec.Command("bash", "-c", "curl -vk https://$(hostname -f)").CombinedOutput()
				GinkgoWriter.Printf("%s\n", out)
				Expect(err).To(Not(HaveOccurred()))
			}

			// Check Rancher version
			operatorVersion, err := kubectl.Run("get", "pod",
				"--namespace", "cattle-system",
				"-l", "app=rancher",
				"-o", "jsonpath={.items[*].status.containerStatuses[*].image}",
			)
			Expect(err).To(Not(HaveOccurred()))
			GinkgoWriter.Printf("Rancher Version:\n%s\n", operatorVersion)
		})

		By("Installing Elemental Operator", func() {
			err := kubectl.RunHelmBinaryWithCustomErr("repo", "add",
				"elemental-operator",
				"https://rancher.github.io/elemental-operator",
			)
			Expect(err).To(Not(HaveOccurred()))

			err = kubectl.RunHelmBinaryWithCustomErr("repo", "update")
			Expect(err).To(Not(HaveOccurred()))

			err = kubectl.RunHelmBinaryWithCustomErr("upgrade", "--install", "elemental-operator",
				"elemental-operator/elemental-operator",
				"--namespace", "cattle-elemental-system",
				"--create-namespace",
			)
			Expect(err).To(Not(HaveOccurred()))

			k.WaitForNamespaceWithPod("cattle-elemental-system", "app=elemental-operator")
			Expect(err).To(Not(HaveOccurred()))

			// Check if an upgrade to a specific version is configured
			if upgradeOperator != "" {
				err = kubectl.RunHelmBinaryWithCustomErr("upgrade", "--install", "elemental-operator",
					upgradeOperator,
					"--namespace", "cattle-elemental-system",
					"--create-namespace",
				)
				Expect(err).To(Not(HaveOccurred()))

				k.WaitForNamespaceWithPod("cattle-elemental-system", "app=elemental-operator")
				Expect(err).To(Not(HaveOccurred()))
			}

			// Check elemental-operator version
			operatorVersion, err := kubectl.Run("get", "pod",
				"--namespace", "cattle-elemental-system",
				"-l", "app=elemental-operator", "-o", "jsonpath={.items[*].status.containerStatuses[*].image}")
			Expect(err).To(Not(HaveOccurred()))
			GinkgoWriter.Printf("Operator Version:\n%s\n", operatorVersion)
		})
	})
})