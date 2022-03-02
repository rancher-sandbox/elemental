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

package managedos

import (
	"strings"

	osv1 "github.com/rancher-sandbox/os2/pkg/apis/rancheros.cattle.io/v1"
	"github.com/rancher-sandbox/os2/pkg/clients"
	upgradev1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func cloudConfig(mos *osv1.ManagedOSImage) ([]byte, error) {
	if mos.Spec.CloudConfig == nil || len(mos.Spec.CloudConfig.Data) == 0 {
		return []byte{}, nil
	}
	data, err := yaml.Marshal(mos.Spec.CloudConfig.Data)
	if err != nil {
		return nil, err
	}
	return append([]byte("#cloud-config\n"), data...), nil
}

func objects(mos *osv1.ManagedOSImage, prefix string) ([]runtime.Object, error) {
	cloudConfig, err := cloudConfig(mos)
	if err != nil {
		return nil, err
	}

	concurrency := int64(1)
	if mos.Spec.Concurrency != nil {
		concurrency = *mos.Spec.Concurrency
	}

	cordon := true
	if mos.Spec.Cordon != nil {
		cordon = *mos.Spec.Cordon
	}

	image := strings.SplitN(mos.Spec.OSImage, ":", 2)
	version := "latest"
	if len(image) == 2 {
		version = image[1]
	}

	selector := mos.Spec.NodeSelector
	if selector == nil {
		selector = &metav1.LabelSelector{}
	}

	return []runtime.Object{
		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: "os-upgrader",
			},
			Rules: []rbacv1.PolicyRule{{
				Verbs:     []string{"update", "get", "list", "watch", "patch"},
				APIGroups: []string{""},
				Resources: []string{"nodes"},
			}},
		},
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "os-upgrader",
			},
			Subjects: []rbacv1.Subject{{
				Kind:      "ServiceAccount",
				Name:      "os-upgrader",
				Namespace: clients.SystemNamespace,
			}},
			RoleRef: rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     "os-upgrader",
			},
		},
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "os-upgrader",
				Namespace: clients.SystemNamespace,
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "os-upgrader-data",
				Namespace: clients.SystemNamespace,
			},
			Data: map[string][]byte{
				"cloud-config": cloudConfig,
			},
		},
		&upgradev1.Plan{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Plan",
				APIVersion: "upgrade.cattle.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "os-upgrader",
				Namespace: clients.SystemNamespace,
			},
			Spec: upgradev1.PlanSpec{
				Concurrency: concurrency,
				Version:     version,
				Tolerations: []corev1.Toleration{{
					Operator: corev1.TolerationOpExists,
				}},
				ServiceAccountName: "os-upgrader",
				NodeSelector:       selector,
				Cordon:             cordon,
				Drain:              mos.Spec.Drain,
				Prepare:            mos.Spec.Prepare,
				Secrets: []upgradev1.SecretSpec{{
					Name: "os-upgrader-data",
					Path: "/run/data",
				}},
				Upgrade: &upgradev1.ContainerSpec{
					Image: PrefixPrivateRegistry(image[0], prefix),
					Command: []string{
						"/usr/sbin/suc-upgrade",
					},
				},
			},
		},
	}, nil
}

func PrefixPrivateRegistry(image, prefix string) string {
	if prefix == "" {
		return image
	}
	return prefix + "/" + image
}
