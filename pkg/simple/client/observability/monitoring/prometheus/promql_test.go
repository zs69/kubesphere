/*
Copyright 2020 KubeSphere Authors

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

package prometheus

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"kubesphere.io/kubesphere/pkg/simple/client/observability/monitoring"
	"kubesphere.io/kubesphere/pkg/simple/client/observability/monitoring/prometheus/testdata"
)

func TestMakeExpr(t *testing.T) {
	tests := []struct {
		name string
		opts monitoring.QueryOptions
	}{
		{
			name: "clusters_cpu_utilisation",
			opts: monitoring.QueryOptions{
				Level: monitoring.LevelCluster,
			},
		},
		{
			name: "cluster_cpu_usage",
			opts: monitoring.QueryOptions{
				Level:       monitoring.LevelCluster,
				ClusterName: "host",
			},
		},
		{
			name: "cluster_cpu_total",
			opts: monitoring.QueryOptions{
				Level:                  monitoring.LevelCluster,
				ClusterResourcesFilter: "host|member",
			},
		},
		{
			name: "node_cpu_utilisation",
			opts: monitoring.QueryOptions{
				Level:       monitoring.LevelNode,
				ClusterName: "host",
				NodeName:    "i-2dazc1d6",
			},
		},
		{
			name: "node_cpu_total",
			opts: monitoring.QueryOptions{
				Level:                  monitoring.LevelNode,
				ClusterResourcesFilter: "host|member",
				NodeResourcesFilter:    "i-2dazc1d6|i-ezjb7gsk",
			},
		},
		{
			name: "workspace_cpu_usage",
			opts: monitoring.QueryOptions{
				Level:         monitoring.LevelWorkspace,
				ClusterName:   "host",
				WorkspaceName: "system-workspace",
			},
		},
		{
			name: "workspace_memory_usage",
			opts: monitoring.QueryOptions{
				Level:                    monitoring.LevelWorkspace,
				ClusterName:              "host",
				WorkspaceResourcesFilter: "system-workspace|demo",
			},
		},
		{
			name: "namespace_cpu_usage",
			opts: monitoring.QueryOptions{
				Level:         monitoring.LevelNamespace,
				ClusterName:   "host",
				WorkspaceName: "system-workspace",
				NamespaceName: "kube-system",
			},
		},
		{
			name: "workload_cpu_usage",
			opts: monitoring.QueryOptions{
				Level:                   monitoring.LevelWorkload,
				WorkloadKind:            "deployment",
				NamespaceName:           "default",
				WorkloadResourcesFilter: "apiserver|coredns",
			},
		},
		{
			name: "workload_deployment_replica_available",
			opts: monitoring.QueryOptions{
				Level:                   monitoring.LevelWorkload,
				WorkloadKind:            ".*",
				NamespaceName:           "default",
				WorkloadResourcesFilter: "apiserver|coredns",
			},
		},
		{
			name: "pod_cpu_usage",
			opts: monitoring.QueryOptions{
				Level:              monitoring.LevelPod,
				NamespaceName:      "default",
				WorkloadKind:       "deployment",
				WorkloadName:       "elasticsearch",
				PodResourcesFilter: "elasticsearch-0",
			},
		},
		{
			name: "pod_memory_usage",
			opts: monitoring.QueryOptions{
				Level:         monitoring.LevelPod,
				NamespaceName: "default",
				PodName:       "elasticsearch-12345",
			},
		},
		{
			name: "pod_memory_usage_wo_cache",
			opts: monitoring.QueryOptions{
				Level:    monitoring.LevelPod,
				NodeName: "i-2dazc1d6",
				PodName:  "elasticsearch-12345",
			},
		},
		{
			name: "pod_net_bytes_transmitted",
			opts: monitoring.QueryOptions{
				Level:                         monitoring.LevelPod,
				NamespacedNameResourcesFilter: "logging/elasticsearch-0|ks/redis",
			},
		},
		{
			name: "container_cpu_usage",
			opts: monitoring.QueryOptions{
				Level:         monitoring.LevelContainer,
				ClusterName:   "host",
				NamespaceName: "default",
				PodName:       "elasticsearch-12345",
				ContainerName: "syscall",
			},
		},
		{
			name: "container_memory_usage",
			opts: monitoring.QueryOptions{
				Level:                  monitoring.LevelContainer,
				NamespaceName:          "default",
				PodName:                "elasticsearch-12345",
				ClusterResourcesFilter: "syscall",
			},
		},
		{
			name: "pvc_inodes_available",
			opts: monitoring.QueryOptions{
				Level:         monitoring.LevelPVC,
				NamespaceName: "default",
				PVCName:       "db-123",
			},
		},
		{
			name: "pvc_inodes_used",
			opts: monitoring.QueryOptions{
				Level:              monitoring.LevelPVC,
				NamespaceName:      "default",
				PVCResourcesFilter: "db-123",
			},
		},
		{
			name: "pvc_inodes_total",
			opts: monitoring.QueryOptions{
				Level:              monitoring.LevelPVC,
				StorageClassName:   "default",
				PVCResourcesFilter: "db-123",
			},
		},
		{
			name: "ingress_request_count",
			opts: monitoring.QueryOptions{
				Level:         monitoring.LevelIngress,
				NamespaceName: "default",
				IngressName:   "ingress-1",
				Job:           "job-1",
				PodName:       "pod-1",
			},
		},
		{
			name: "etcd_server_list",
			opts: monitoring.QueryOptions{
				Level:       monitoring.LevelComponent,
				ClusterName: "host",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected := testdata.PromQLs[tt.name]
			result := makeExpr(tt.name, tt.opts)
			if diff := cmp.Diff(result, expected); diff != "" {
				t.Fatalf("%T differ (-got, +want): %s", expected, diff)
			}
		})
	}
}
