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

package testdata

var PromQLs = map[string]string{
	"clusters_cpu_utilisation":              `round(sum(sum by (cluster)(node:node_num_cpu:sum{}) * on(cluster):node_cpu_utilisation:avg1m{})/sum(node:node_num_cpu:sum{}),0.001)`,
	"cluster_cpu_usage":                     `sum by (cluster)(node:node_num_cpu:sum{cluster="host"}) * on(cluster):node_cpu_utilisation:avg1m{cluster="host"}`,
	"cluster_cpu_total":                     `sum by (cluster)(node:node_num_cpu:sum{cluster=~"host|member"})`,
	"node_cpu_utilisation":                  `node:node_cpu_utilisation:avg1m{cluster="host",node="i-2dazc1d6"}`,
	"node_cpu_total":                        `node:node_num_cpu:sum{cluster=~"host|member",node=~"i-2dazc1d6|i-ezjb7gsk"}`,
	"workspace_cpu_usage":                   `round(sum by (cluster, workspace) (namespace:container_cpu_usage_seconds_total:sum_rate{namespace!="", cluster="host",workspace="system-workspace"}), 0.001)`,
	"workspace_memory_usage":                `sum by (cluster, workspace) (namespace:container_memory_usage_bytes:sum{namespace!="", cluster="host",workspace=~"system-workspace|demo"})`,
	"namespace_cpu_usage":                   `round(namespace:container_cpu_usage_seconds_total:sum_rate{namespace!="", cluster="host",workspace="system-workspace",namespace="kube-system"}, 0.001)`,
	"workload_cpu_usage":                    `round(namespace:workload_cpu_usage:sum{namespace="default",workload=~"Deployment:(apiserver|coredns)"}, 0.001)`,
	"workload_deployment_replica_available": `label_join(sum (label_join(label_replace(kube_deployment_status_replicas_available{namespace="default",deployment!="", deployment=~"apiserver|coredns"}, "owner_kind", "Deployment", "", ""), "workload", "", "deployment")) by (namespace, owner_kind, workload), "workload", ":", "owner_kind", "workload")`,
	"pod_cpu_usage":                         `round(sum by (cluster,namespace, pod) (irate(container_cpu_usage_seconds_total{job="kubelet", pod!="", image!=""}[5m])) * on (cluster, namespace, pod) group_left (qos,owner_kind,owner_name,node) qos_owner_node:kube_pod_info:{namespace="default",owner_kind="ReplicaSet", owner_name=~"^elasticsearch-[^-]{1,10}$",pod=~"elasticsearch-0"},0.001)`,
	"pod_memory_usage":                      `sum by (cluster, namespace, pod) (container_memory_usage_bytes{job="kubelet", pod!="", image!=""}) * on (cluster,namespace, pod) group_left (qos,owner_kind,owner_name,node) qos_owner_node:kube_pod_info:{namespace="default",pod="elasticsearch-12345"}`,
	"pod_memory_usage_wo_cache":             `sum by (cluster, namespace, pod) (container_memory_working_set_bytes{job="kubelet", pod!="", image!=""})  * on (cluster, namespace, pod) group_left (qos,owner_kind,owner_name,node) qos_owner_node:kube_pod_info:{node="i-2dazc1d6",pod="elasticsearch-12345"}`,
	"pod_net_bytes_transmitted":             `round(sum by (cluster, namespace, pod) (irate(container_network_transmit_bytes_total{pod!="", interface!~"^(cali.+|tunl.+|dummy.+|kube.+|flannel.+|cni.+|docker.+|veth.+|lo.*)", job="kubelet"}[5m])) * on (cluster, namespace, pod) group_left (qos,owner_kind,owner_name,node) qos_owner_node:kube_pod_info:{namespace=~"logging|ks|",pod=~"elasticsearch-0|redis|"}, 0.001)`,
	"container_cpu_usage":                   `round(sum by (cluster, namespace, pod, container) (irate(container_cpu_usage_seconds_total{job="kubelet", container!="POD", container!="", image!="", cluster="host",namespace="default",pod="elasticsearch-12345",container="syscall"}[5m])), 0.001)`,
	"container_memory_usage":                `sum by (cluster, namespace, pod, container) (container_memory_usage_bytes{job="kubelet", container!="POD", container!="", image!="", cluster=~"syscall",namespace="default",pod="elasticsearch-12345"})`,
	"pvc_inodes_available":                  `max by (cluster, namespace, persistentvolumeclaim) (kubelet_volume_stats_inodes_free) * on (cluster, namespace, persistentvolumeclaim) group_left (storageclass) kube_persistentvolumeclaim_info{namespace="default",persistentvolumeclaim="db-123"}`,
	"pvc_inodes_used":                       `max by (cluster, namespace, persistentvolumeclaim) (kubelet_volume_stats_inodes_used) * on (cluster, namespace, persistentvolumeclaim) group_left (storageclass) kube_persistentvolumeclaim_info{namespace="default",persistentvolumeclaim=~"db-123"}`,
	"pvc_inodes_total":                      `max by (cluster, namespace, persistentvolumeclaim) (kubelet_volume_stats_inodes) * on (cluster, namespace, persistentvolumeclaim) group_left (storageclass) kube_persistentvolumeclaim_info{storageclass="default",persistentvolumeclaim=~"db-123"}`,
	"ingress_request_count":                 `round(sum by(cluster)(increase(nginx_ingress_controller_requests{exported_namespace="default", ingress="ingress-1",job="job-1",controller_pod="pod-1"}[5m])))`,
	"etcd_server_list":                      `label_replace(up{job="etcd", cluster="host"}, "node_ip", "$1", "instance", "(.*):.*")`,
}
