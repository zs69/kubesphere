/*
Copyright 2021 The KubeSphere Authors.

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

package utils

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"

	"kubesphere.io/kubesphere/pkg/constants"
)

// ClusterRole get the cluster role of the kubernetes cluster.
// If the current is a host cluster or a cluster with multi-cluster mode disabled, this controller should update the status of license.
// If the current is a member cluster, just return.
// TODO: We should use a more reliable method to find out whether this cluster is a host cluster or not.
func ClusterRole(ctx context.Context, config *rest.Config) (string, error) {
	dyn, err := dynamic.NewForConfig(config)
	if err != nil {
		return "", nil
	}

	obj, err := dyn.Resource(schema.GroupVersionResource{
		Group:    "installer.kubesphere.io",
		Version:  "v1alpha1",
		Resource: "clusterconfigurations",
	}).Namespace(constants.KubeSphereNamespace).
		Get(ctx, "ks-installer", metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	val := obj.Object["spec"]
	if val != nil {
		spec := val.(map[string]interface{})
		if val, exists := spec["multicluster"]; exists {
			m := val.(map[string]interface{})
			if m["clusterRole"] == "member" {
				return "member", err
			} else if m["clusterRole"] == "host" {
				return "host", err
			}
		}
	}

	return "", nil
}
