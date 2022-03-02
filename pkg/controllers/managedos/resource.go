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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"
	"github.com/rancher/wrangler/pkg/gvk"
	"github.com/rancher/wrangler/pkg/name"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
)

func ToResources(objs []runtime.Object) (result []v1alpha1.BundleResource, err error) {
	for _, obj := range objs {
		obj = obj.DeepCopyObject()
		if err := gvk.Set(obj); err != nil {
			return nil, fmt.Errorf("failed to set gvk: %w", err)
		}

		typeMeta, err := meta.TypeAccessor(obj)
		if err != nil {
			return nil, err
		}

		meta, err := meta.Accessor(obj)
		if err != nil {
			return nil, err
		}

		data, err := json.Marshal(obj)
		if err != nil {
			return nil, err
		}

		digest := sha256.Sum256(data)
		filename := name.SafeConcatName(typeMeta.GetKind(), meta.GetNamespace(), meta.GetName(), hex.EncodeToString(digest[:])[:12]) + ".yaml"
		result = append(result, v1alpha1.BundleResource{
			Name:    filename,
			Content: string(data),
		})
	}
	return
}
