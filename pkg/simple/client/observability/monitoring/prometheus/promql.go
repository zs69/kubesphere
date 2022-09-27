/*
Copyright 2019 The KubeSphere Authors.
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
	"fmt"
	"strings"

	"kubesphere.io/kubesphere/pkg/constants"
	"kubesphere.io/kubesphere/pkg/simple/client/observability/monitoring"
)

const (
	StatefulSet = "StatefulSet"
	DaemonSet   = "DaemonSet"
	Deployment  = "Deployment"
)

var promQLTemplates = map[string]string{
	// clusters
	"clusters_cpu_utilisation":           `round(sum(sum by (cluster)(node:node_num_cpu:sum{$filter}) * on(cluster):node_cpu_utilisation:avg1m{$filter})/sum(node:node_num_cpu:sum{$filter}),0.001)`,
	"clusters_cpu_usage":                 `round(sum(sum by (cluster)(node:node_num_cpu:sum{$filter}) * on(cluster):node_cpu_utilisation:avg1m{$filter}),0.001)`,
	"clusters_cpu_total":                 `sum(node:node_num_cpu:sum{$filter})`,
	"clusters_memory_utilisation":        `round(1- sum(node:node_memory_bytes_available:sum{$filter})/sum(node:node_memory_bytes_total:sum{$filter}),0.001)`,
	"clusters_memory_available":          "sum(node:node_memory_bytes_available:sum{$filter})",
	"clusters_memory_total":              "sum(node:node_memory_bytes_total:sum{$filter})",
	"clusters_memory_usage_wo_cache":     "sum(node:node_memory_bytes_total:sum{$filter}) - sum(node:node_memory_bytes_available:sum{$filter})",
	"clusters_disk_size_usage":           `sum(max(node_filesystem_size_bytes{device=~"/dev/.*", device!~"/dev/loop\\d+", job="node-exporter",$filter} - node_filesystem_avail_bytes{device=~"/dev/.*", device!~"/dev/loop\\d+", job="node-exporter",$filter}) by (device, instance))`,
	"clusters_disk_size_utilisation":     `1 - sum(max by(device, instance) (node_filesystem_avail_bytes{device!~"/dev/loop\\d+",device=~"/dev/.*",job="node-exporter",$filter})) / sum(max by(device, instance) (node_filesystem_size_bytes{device!~"/dev/loop\\d+",device=~"/dev/.*",job="node-exporter",$filter}))`,
	"clusters_disk_size_capacity":        `sum(max(node_filesystem_size_bytes{device=~"/dev/.*", device!~"/dev/loop\\d+", job="node-exporter",$filter}) by (device, instance))`,
	"clusters_disk_size_available":       `sum(max(node_filesystem_avail_bytes{device=~"/dev/.*", device!~"/dev/loop\\d+", job="node-exporter",$filter}) by (device, instance))`,
	"clusters_pod_cpu_usage":             `round(sum (irate(container_cpu_usage_seconds_total{job="kubelet", container!="POD", container!="", image!=""}[5m]) and on(namespace,pod)node_namespace_pod:kube_pod_info:{}), 0.001)`,
	"clusters_pod_cpu_requests_total":    `sum(sum by(node) (kube_pod_container_resource_requests{job="kube-state-metrics",resource="cpu"}) * on(node) group_left(host_ip, role) max by(node, host_ip, role) (node_namespace_pod:kube_pod_info:{host_ip!="",node!=""}))`,
	"clusters_pod_cpu_limits_total":      `sum(sum by(node) (kube_pod_container_resource_limits{job="kube-state-metrics",resource="cpu"}) * on(node) group_left(host_ip, role) max by(node, host_ip, role) (node_namespace_pod:kube_pod_info:{host_ip!="",node!=""}))`,
	"clusters_pod_memory_requests_total": `sum(sum by(node) (kube_pod_container_resource_requests{job="kube-state-metrics",resource="memory"}) * on(node) group_left(host_ip, role) max by(node, host_ip, role) (node_namespace_pod:kube_pod_info:{host_ip!="",node!=""}))`,
	"clusters_pod_memory_limits_total":   `sum(sum by(node) (kube_pod_container_resource_limits{job="kube-state-metrics",resource="memory"}) * on(node) group_left(host_ip, role) max by(node, host_ip, role) (node_namespace_pod:kube_pod_info:{host_ip!="",node!=""}))`,
	"clusters_pod_count":                 `count(kube_pod_info{job="kube-state-metrics",$filter})`,
	"clusters_pod_quota":                 `sum(max by (cluster,node)(kube_node_status_capacity{resource="pods",$filter})  unless on (cluster,node) (kube_node_status_condition{condition="Ready",status=~"unknown|false",$filter} > 0))`,
	"clusters_pod_utilisation":           `count(kube_pod_info{job="kube-state-metrics",$filter})/sum(max by (cluster,node)(kube_node_status_capacity{resource="pods",$filter})  unless on (cluster,node) (kube_node_status_condition{condition="Ready",status=~"unknown|false",$filter} > 0))`,
	"clusters_pod_running_count":         `count(kube_pod_info{job="kube-state-metrics",$filter} and on(pod, namespace) (kube_pod_status_phase{job="kube-state-metrics",phase="Running",$filter} > 0))`,
	"clusters_pod_succeeded_count":       `count(kube_pod_info{job="kube-state-metrics",$filter} and on(pod, namespace) (kube_pod_status_phase{job="kube-state-metrics",phase="Succeeded",$filter} > 0))`,
	"clusters_pod_pending_count":         `count(kube_pod_info{job="kube-state-metrics",$filter} and on(pod, namespace) (kube_pod_status_phase{job="kube-state-metrics",phase="Pending",$filter} > 0))`,
	"clusters_pod_failed_count":          `count(kube_pod_info{job="kube-state-metrics",$filter} and on(pod, namespace) (kube_pod_status_phase{job="kube-state-metrics",phase="Failed",$filter} > 0))`,
	"clusters_pod_unknown_count":         `count(kube_pod_info{job="kube-state-metrics",$filter} and on(pod, namespace) (kube_pod_status_phase{job="kube-state-metrics",phase="Unknown",$filter} > 0))`,
	"clusters_pod_oomkilled_count":       `sum(kube_pod_container_status_last_terminated_reason{reason="OOMKilled",$filter})`,
	"clusters_pod_evicted_count":         `sum(kube_pod_status_reason{reason="Evicted",$filter}>0)`,
	"clusters_pod_qos_guaranteed_count":  `count (qos_owner_node:kube_pod_info:{qos="guaranteed",$filter})`,
	"clusters_pod_qos_burstable_count":   `count (qos_owner_node:kube_pod_info:{qos="burstable", $filter})`,
	"clusters_pod_qos_besteffort_count":  `count (qos_owner_node:kube_pod_info:{qos="besteffort",$filter})`,
	"clusters_namespace_count":           `sum(kube_namespace_labels{$filter})`,
	"clusters_count":                     `count(sum by (cluster)(kube_node_labels{$filter}))`,
	"clusters_node_count":                `sum(kube_node_labels{$filter})`,
	"clusters_cronjob_count":             `sum(kube_cronjob_labels{$filter})`,
	"clusters_pvc_count":                 `sum(kube_persistentvolumeclaim_info{$filter})`,
	"clusters_daemonset_count":           `sum(kube_daemonset_labels{$filter})`,
	"clusters_deployment_count":          `sum(kube_deployment_labels{$filter})`,
	"clusters_endpoint_count":            `sum(kube_endpoint_labels{$filter})`,
	"clusters_hpa_count":                 `sum(kube_horizontalpodautoscaler_labels{$filter})`,
	"clusters_job_count":                 `sum(kube_job_labels{$filter})`,
	"clusters_statefulset_count":         `sum(kube_statefulset_labels{$filter})`,
	"clusters_replicaset_count":          `sum(kube_replicaset_labels{$filter})`,
	"clusters_service_count":             `sum(kube_service_info{$filter})`,
	"clusters_secret_count":              `sum(kube_secret_info{$filter})`,
	"clusters_pv_count":                  `sum(kube_persistentvolume_labels{$filter})`,
	"clusters_ingresses_count":           `sum(kube_ingress_labels{$filter})`,
	"clusters_gpu_utilization":           `round(avg(DCGM_FI_PROF_GR_ENGINE_ACTIVE{$filter}) / 100, 0.00001) or round(avg(DCGM_FI_DEV_GPU_UTIL{$filter}) / 100, 0.00001)`,
	"clusters_gpu_usage":                 `round(sum(DCGM_FI_PROF_GR_ENGINE_ACTIVE{$filter}) / 100, 0.00001) or round(sum(DCGM_FI_DEV_GPU_UTIL{$filter}) / 100, 0.00001)`,
	"clusters_gpu_total":                 `sum(kube_node_status_capacity{resource="nvidia_com_gpu",$filter})`,
	"clusters_gpu_memory_utilization":    `sum(DCGM_FI_DEV_FB_USED{$filter}) / sum(DCGM_FI_DEV_FB_FREE{$filter} + DCGM_FI_DEV_FB_USED{$filter})`,
	"clusters_gpu_memory_usage":          `sum(DCGM_FI_DEV_FB_USED{$filter}) * 1024 * 1024`,
	"clusters_gpu_memory_available":      `sum(DCGM_FI_DEV_FB_FREE{$filter}) * 1024 * 1024`,
	"clusters_gpu_memory_total":          `sum(DCGM_FI_DEV_FB_FREE{$filter} + DCGM_FI_DEV_FB_USED{$filter}) * 1024 * 1024`,
	"clusters_alerts_critical_total":     `sum(ALERTS{severity="critical",$filter})`,
	"clusters_alerts_warning_total":      `sum(ALERTS{severity="warning",$filter})`,
	"clusters_alerts_info_total":         `sum(ALERTS{severity="info",$filter})`,

	// cluster
	"cluster_cpu_utilisation":                            ":node_cpu_utilisation:avg1m{$filter}",
	"cluster_cpu_usage":                                  `sum by (cluster)(node:node_num_cpu:sum{$filter}) * on(cluster):node_cpu_utilisation:avg1m{$filter}`,
	"cluster_cpu_total":                                  "sum by (cluster)(node:node_num_cpu:sum{$filter})",
	"cluster_cpu_non_master_total":                       `sum by (cluster)(node:node_num_cpu:sum{role!="master",$filter})`,
	"cluster_memory_utilisation":                         ":node_memory_utilisation:{$filter}",
	"cluster_memory_available":                           "sum by (cluster)(node:node_memory_bytes_available:sum{$filter})",
	"cluster_memory_total":                               "sum by (cluster)(node:node_memory_bytes_total:sum{$filter})",
	"cluster_memory_non_master_total":                    `sum by (cluster)(node:node_memory_bytes_total:sum{role!="master", $filter})`,
	"cluster_memory_usage_wo_cache":                      "sum by (cluster)(node:node_memory_bytes_total:sum{$filter}) - sum by (cluster)(node:node_memory_bytes_available:sum{$filter})",
	"cluster_net_utilisation":                            ":node_net_utilisation:sum_irate{$filter}",
	"cluster_net_bytes_transmitted":                      "sum by (cluster)(node:node_net_bytes_transmitted:sum_irate{$filter})",
	"cluster_net_bytes_received":                         "sum by (cluster)(node:node_net_bytes_received:sum_irate{$filter})",
	"cluster_disk_read_iops":                             "sum by (cluster)(node:data_volume_iops_reads:sum{$filter})",
	"cluster_disk_write_iops":                            "sum by (cluster)(node:data_volume_iops_writes:sum{$filter})",
	"cluster_disk_read_throughput":                       "sum by (cluster)(node:data_volume_throughput_bytes_read:sum{$filter})",
	"cluster_disk_write_throughput":                      "sum by (cluster)(node:data_volume_throughput_bytes_written:sum{$filter})",
	"cluster_disk_size_usage":                            `sum by (cluster)(max by (cluster, device, instance)(node_filesystem_size_bytes{device=~"/dev/.*", device!~"/dev/loop\\d+", job="node-exporter",$filter} - node_filesystem_avail_bytes{device=~"/dev/.*", device!~"/dev/loop\\d+", job="node-exporter",$filter}))`,
	"cluster_disk_size_utilisation":                      `1 - sum(max by(cluster, device, instance) (node_filesystem_avail_bytes{device!~"/dev/loop\\d+",device=~"/dev/.*",job="node-exporter",$filter})) / sum(max by(device, instance) (node_filesystem_size_bytes{device!~"/dev/loop\\d+",device=~"/dev/.*",job="node-exporter",$filter}))`,
	"cluster_disk_size_capacity":                         `sum by (cluster)(max(node_filesystem_size_bytes{device=~"/dev/.*", device!~"/dev/loop\\d+", job="node-exporter",$filter}) by (device, instance))`,
	"cluster_disk_size_available":                        `sum by (cluster)(max(node_filesystem_avail_bytes{device=~"/dev/.*", device!~"/dev/loop\\d+", job="node-exporter",$filter}) by (device, instance))`,
	"cluster_disk_inode_total":                           `sum by (cluster)(node:node_inodes_total:)`,
	"cluster_disk_inode_usage":                           `sum by (cluster)(node:node_inodes_total:) - sum(node:node_inodes_free:)`,
	"cluster_disk_inode_utilisation":                     `cluster:disk_inode_utilization:ratio`,
	"cluster_pod_count":                                  `count by(cluster)(kube_pod_info{job="kube-state-metrics",$filter})`,
	"cluster_pod_quota":                                  `sum by(cluster)(max by (cluster,node)(kube_node_status_capacity{resource="pods",$filter})  unless on (cluster,node) (kube_node_status_condition{condition="Ready",status=~"unknown|false",$filter} > 0))`,
	"cluster_pod_utilisation":                            `count by(cluster)(kube_pod_info{job="kube-state-metrics",$filter})/sum(max by (cluster,node)(kube_node_status_capacity{resource="pods",$filter})  unless on (cluster,node) (kube_node_status_condition{condition="Ready",status=~"unknown|false",$filter} > 0))`,
	"cluster_pod_running_count":                          `count by(cluster)(kube_pod_info{job="kube-state-metrics",$filter} and on(pod, namespace) (kube_pod_status_phase{job="kube-state-metrics",phase="Running",$filter} > 0))`,
	"cluster_pod_succeeded_count":                        `count by(cluster)(kube_pod_info{job="kube-state-metrics",$filter} and on(pod, namespace) (kube_pod_status_phase{job="kube-state-metrics",phase="Succeeded",$filter} > 0))`,
	"cluster_pod_pending_count":                          `count by(cluster)(kube_pod_info{job="kube-state-metrics",$filter} and on(pod, namespace) (kube_pod_status_phase{job="kube-state-metrics",phase="Pending",$filter} > 0))`,
	"cluster_pod_failed_count":                           `count by(cluster)(kube_pod_info{job="kube-state-metrics",$filter} and on(pod, namespace) (kube_pod_status_phase{job="kube-state-metrics",phase="Failed",$filter} > 0))`,
	"cluster_pod_unknown_count":                          `count by(cluster)(kube_pod_info{job="kube-state-metrics",$filter} and on(pod, namespace) (kube_pod_status_phase{job="kube-state-metrics",phase="Unknown",$filter} > 0))`,
	"cluster_pod_abnormal_count":                         `cluster:pod_abnormal:sum`,
	"cluster_pod_oomkilled_count":                        `sum(kube_pod_container_status_last_terminated_reason{reason="OOMKilled",$filter})`,
	"cluster_pod_evicted_count":                          `sum by(cluster)(kube_pod_status_reason{reason="Evicted",$filter}>0)`,
	"cluster_pod_qos_guaranteed_count":                   `count  by(cluster)(qos_owner_node:kube_pod_info:{qos="guaranteed",$filter})`,
	"cluster_pod_qos_burstable_count":                    `count  by(cluster)(qos_owner_node:kube_pod_info:{qos="burstable", $filter})`,
	"cluster_pod_qos_besteffort_count":                   `count  by(cluster)(qos_owner_node:kube_pod_info:{qos="besteffort",$filter})`,
	"cluster_pod_cpu_usage":                              `round(sum by(cluster) (irate(container_cpu_usage_seconds_total{job="kubelet", container!="POD", container!="", image!=""}[5m]) and on(namespace,pod)node_namespace_pod:kube_pod_info:{}), 0.001)`,
	"cluster_pod_cpu_non_master_usage":                   `round(sum by(cluster)(irate(container_cpu_usage_seconds_total{job="kubelet", container!="POD", container!="", image!=""}[5m]) and on(namespace,pod)node_namespace_pod:kube_pod_info:{role!="master"}), 0.001)`,
	"cluster_pod_cpu_requests_total":                     `sum by(cluster)(sum by(node) (kube_pod_container_resource_requests{job="kube-state-metrics",resource="cpu"}) * on(node) group_left(host_ip, role) max by(node, host_ip, role) (node_namespace_pod:kube_pod_info:{host_ip!="",node!=""}))`,
	"cluster_pod_cpu_requests_non_master_total":          `sum by(cluster)(sum by(node) (kube_pod_container_resource_requests{job="kube-state-metrics",resource="cpu"}) * on(node) group_left(host_ip, role) max by(node, host_ip, role) (node_namespace_pod:kube_pod_info:{host_ip!="",node!="",role!="master"}))`,
	"cluster_pod_cpu_limits_total":                       `sum by(cluster)(sum by(node) (kube_pod_container_resource_limits{job="kube-state-metrics",resource="cpu"}) * on(node) group_left(host_ip, role) max by(node, host_ip, role) (node_namespace_pod:kube_pod_info:{host_ip!="",node!=""}))`,
	"cluster_pod_cpu_limits_non_master_total":            `sum by(cluster)(sum by(node) (kube_pod_container_resource_limits{job="kube-state-metrics",resource="cpu"}) * on(node) group_left(host_ip, role) max by(node, host_ip, role) (node_namespace_pod:kube_pod_info:{host_ip!="",node!="",role!="master"}))`,
	"cluster_pod_memory_usage_wo_cache":                  `sum by(cluster)(sum by(node) (container_memory_working_set_bytes{job="kubelet", pod!="", image!=""}and on(namespace,pod)node_namespace_pod:kube_pod_info:{}))`,
	"cluster_pod_memory_non_master_usage_wo_cache":       `sum by(cluster)(sum by(node) (container_memory_working_set_bytes{job="kubelet", pod!="", image!=""}and on(namespace,pod)node_namespace_pod:kube_pod_info:{role!="master"}))`,
	"cluster_pod_memory_requests_total":                  `sum by(cluster)(sum by(node) (kube_pod_container_resource_requests{job="kube-state-metrics",resource="memory"}) * on(node) group_left(host_ip, role) max by(node, host_ip, role) (node_namespace_pod:kube_pod_info:{host_ip!="",node!=""}))`,
	"cluster_pod_memory_requests_non_master_total":       `sum by(cluster)(sum by(node) (kube_pod_container_resource_requests{job="kube-state-metrics",resource="memory"}) * on(node) group_left(host_ip, role) max by(node, host_ip, role) (node_namespace_pod:kube_pod_info:{host_ip!="",node!="",role!="master"}))`,
	"cluster_pod_memory_limits_total":                    `sum by(cluster)(sum by(node) (kube_pod_container_resource_limits{job="kube-state-metrics",resource="memory"}) * on(node) group_left(host_ip, role) max by(node, host_ip, role) (node_namespace_pod:kube_pod_info:{host_ip!="",node!=""}))`,
	"cluster_pod_memory_limits_non_master_total":         `sum by(cluster)(sum by(node) (kube_pod_container_resource_limits{job="kube-state-metrics",resource="memory"}) * on(node) group_left(host_ip, role) max by(node, host_ip, role)(node_namespace_pod:kube_pod_info:{host_ip!="",node!="",role!="master"}))`,
	"cluster_namespace_quota_cpu_usage":                  `sum by(cluster)(namespace:container_cpu_usage_seconds_total:sum_rate and on(namespace) kube_resourcequota)`,
	"cluster_namespace_quota_cpu_requests_hard_total":    `sum by(cluster)(kube_resourcequota{type="hard",resource="requests.cpu"})`,
	"cluster_namespace_quota_cpu_limits_hard_total":      `sum by(cluster)(kube_resourcequota{type="hard",resource="limits.cpu"})`,
	"cluster_namespace_quota_memory_usage":               `sum by(cluster)(namespace:container_memory_usage_bytes:sum and on(namespace) kube_resourcequota)`,
	"cluster_namespace_quota_memory_usage_wo_cache":      `sum by(cluster)(namespace:container_memory_usage_bytes_wo_cache:sum and on(namespace) kube_resourcequota)`,
	"cluster_namespace_quota_memory_requests_hard_total": `sum by(cluster)(kube_resourcequota{type="hard",resource="requests.memory"})`,
	"cluster_namespace_quota_memory_limits_hard_total":   `sum by(cluster)(kube_resourcequota{type="hard",resource="limits.memory", $filter})`,

	"cluster_namespace_count":            `sum by (cluster)(kube_namespace_labels{$filter})`,
	"cluster_node_online":                `sum by (cluster)(kube_node_status_condition{condition="Ready",status="true",$filter})`,
	"cluster_node_offline":               `cluster:node_offline:sum{$filter}`,
	"cluster_node_total":                 `sum by (cluster)(kube_node_status_condition{condition="Ready", $filter})`,
	"cluster_cronjob_count":              `sum by (cluster)(kube_cronjob_labels{$filter})`,
	"cluster_pvc_count":                  `sum by (cluster)(kube_persistentvolumeclaim_info{$filter})`,
	"cluster_daemonset_count":            `sum by (cluster)(kube_daemonset_labels{$filter})`,
	"cluster_deployment_count":           `sum by (cluster)(kube_deployment_labels{$filter})`,
	"cluster_endpoint_count":             `sum by (cluster)(kube_endpoint_labels{$filter})`,
	"cluster_hpa_count":                  `sum by (cluster)(kube_horizontalpodautoscaler_labels{$filter})`,
	"cluster_job_count":                  `sum by (cluster)(kube_job_labels{$filter})`,
	"cluster_statefulset_count":          `sum by (cluster)(kube_statefulset_labels{$filter})`,
	"cluster_replicaset_count":           `count by (cluster)(kube_replicaset_labels{$filter})`,
	"cluster_service_count":              `sum by (cluster)(kube_service_info{$filter})`,
	"cluster_secret_count":               `sum by (cluster)(kube_secret_info{$filter})`,
	"cluster_pv_count":                   `sum by (cluster)(kube_persistentvolume_labels{$filter})`,
	"cluster_ingresses_extensions_count": `sum by (cluster)(kube_ingress_labels{$filter})`,
	"cluster_load1":                      `sum by (cluster)(node_load1{job="node-exporter",$filter}) / sum(node:node_num_cpu:sum)`,
	"cluster_load5":                      `sum by (cluster)(node_load5{job="node-exporter",$filter}) / sum(node:node_num_cpu:sum)`,
	"cluster_load15":                     `sum by (cluster)(node_load15{job="node-exporter",$filter}) / sum(node:node_num_cpu:sum)`,
	"cluster_pod_abnormal_ratio":         `cluster:pod_abnormal:ratio{$filter}`,
	"cluster_node_offline_ratio":         `cluster:node_offline:ratio{$filter}`,
	"cluster_gpu_utilization":            `round(avg by (cluster)(DCGM_FI_PROF_GR_ENGINE_ACTIVE{$filter}) / 100, 0.00001) or round(avg by (cluster)(DCGM_FI_DEV_GPU_UTIL{$filter}) / 100, 0.00001)`,
	"cluster_gpu_usage":                  `round(sum by (cluster)(DCGM_FI_PROF_GR_ENGINE_ACTIVE{$filter}) / 100, 0.00001) or round(sum by (cluster)(DCGM_FI_DEV_GPU_UTIL{$filter}) / 100, 0.00001)`,
	"cluster_gpu_total":                  `sum by (cluster)(kube_node_status_capacity{resource="nvidia_com_gpu",$filter})`,
	"cluster_gpu_memory_utilization":     `sum by (cluster)(DCGM_FI_DEV_FB_USED{$filter}) / sum by(cluster)(DCGM_FI_DEV_FB_FREE{$filter} + DCGM_FI_DEV_FB_USED{$filter})`,
	"cluster_gpu_memory_usage":           `sum by (cluster)(DCGM_FI_DEV_FB_USED{$filter}) * 1024 * 1024`,
	"cluster_gpu_memory_available":       `sum by (cluster)(DCGM_FI_DEV_FB_FREE{$filter}) * 1024 * 1024`,
	"cluster_gpu_memory_total":           `sum by (cluster)(DCGM_FI_DEV_FB_FREE{$filter} + DCGM_FI_DEV_FB_USED{$filter}) * 1024 * 1024`,

	// node
	"node_cpu_utilisation":        "node:node_cpu_utilisation:avg1m{$filter}",
	"node_cpu_total":              "node:node_num_cpu:sum{$filter}",
	"node_memory_utilisation":     "node:node_memory_utilisation:{$filter}",
	"node_memory_available":       "node:node_memory_bytes_available:sum{$filter}",
	"node_memory_total":           "node:node_memory_bytes_total:sum{$filter}",
	"node_memory_usage_wo_cache":  "node:node_memory_bytes_total:sum{$filter} - node:node_memory_bytes_available:sum{$filter}",
	"node_net_utilisation":        "node:node_net_utilisation:sum_irate{$filter}",
	"node_net_bytes_transmitted":  "node:node_net_bytes_transmitted:sum_irate{$filter}",
	"node_net_bytes_received":     "node:node_net_bytes_received:sum_irate{$filter}",
	"node_disk_read_iops":         "node:data_volume_iops_reads:sum{$filter}",
	"node_disk_write_iops":        "node:data_volume_iops_writes:sum{$filter}",
	"node_disk_read_throughput":   "node:data_volume_throughput_bytes_read:sum{$filter}",
	"node_disk_write_throughput":  "node:data_volume_throughput_bytes_written:sum{$filter}",
	"node_disk_size_capacity":     `sum by (cluster, node)(max by (cluster, device, node)(node_filesystem_size_bytes{device=~"/dev/.*", device!~"/dev/loop\\d+", job="node-exporter"} * on (cluster, namespace, pod) group_left(node) node_namespace_pod:kube_pod_info:{$filter}))`,
	"node_disk_size_available":    `node:disk_space_available:{$filter}`,
	"node_disk_size_usage":        `sum by (cluster, node)(max by (cluster, device, node)((node_filesystem_size_bytes{device=~"/dev/.*", device!~"/dev/loop\\d+", job="node-exporter"} - node_filesystem_avail_bytes{device=~"/dev/.*", device!~"/dev/loop\\d+", job="node-exporter"}) * on (cluster, namespace, pod) group_left(node) node_namespace_pod:kube_pod_info:{$filter})) `,
	"node_disk_size_utilisation":  `node:disk_space_utilization:ratio{$filter}`,
	"node_disk_inode_total":       `node:node_inodes_total:{$filter}`,
	"node_disk_inode_usage":       `node:node_inodes_total:{$filter} - node:node_inodes_free:{$filter}`,
	"node_disk_inode_utilisation": `node:disk_inode_utilization:ratio{$filter}`,
	"node_pod_count":              `node:pod_count:sum{$filter}`,
	"node_pod_quota":              `max by (cluster, node)(kube_node_status_capacity{resource="pods",$filter})  unless on (cluster, node) (kube_node_status_condition{condition="Ready",status=~"unknown|false"} > 0)`,
	"node_pod_utilisation":        `node:pod_utilization:ratio{$filter}`,
	"node_pod_running_count":      `node:pod_running:count{$filter}`,
	"node_pod_succeeded_count":    `node:pod_succeeded:count{$filter}`,
	"node_pod_abnormal_count":     `node:pod_abnormal:count{$filter}`,
	"node_cpu_usage":              `round(node:node_cpu_utilisation:avg1m{$filter} * node:node_num_cpu:sum{$filter}, 0.001)`,
	"node_load1":                  `node:load1:ratio{$filter}`,
	"node_load5":                  `node:load5:ratio{$filter}`,
	"node_load15":                 `node:load15:ratio{$filter}`,
	"node_pod_abnormal_ratio":     `node:pod_abnormal:ratio{$filter}`,
	"node_pleg_quantile":          `node_quantile:kubelet_pleg_relist_duration_seconds:histogram_quantile{$filter}`,
	"node_gpu_utilization":        `round(avg(DCGM_FI_PROF_GR_ENGINE_ACTIVE * on (cluster, namespace, pod) group_left(node, host_ip, role) node_namespace_pod:kube_pod_info:{$filter}) by(node) / 100, 0.00001) or round(avg(DCGM_FI_DEV_GPU_UTIL* on (cluster, namespace, pod) group_left(node, host_ip, role) node_namespace_pod:kube_pod_info:{$filter}) by(cluster, node)/ 100, 0.00001)`,
	"node_gpu_usage":              `round(sum(DCGM_FI_PROF_GR_ENGINE_ACTIVE * on (cluster, namespace, pod) group_left(node, host_ip, role) node_namespace_pod:kube_pod_info:{$filter}) by(node) / 100, 0.00001) or round(sum(DCGM_FI_DEV_GPU_UTIL* on (cluster, namespace, pod) group_left(node, host_ip, role) node_namespace_pod:kube_pod_info:{$filter}) by(cluster, node)/ 100, 0.00001)`,
	"node_gpu_total":              `kube_node_status_capacity{resource="nvidia_com_gpu", $filter}`,
	"node_gpu_memory_utilization": `avg(DCGM_FI_DEV_FB_USED/(DCGM_FI_DEV_FB_FREE + DCGM_FI_DEV_FB_USED) * on(cluster, namespace , pod) group_left(node) (node_namespace_pod:kube_pod_info:{$filter})) by(node)`,
	"node_gpu_memory_usage":       `DCGM_FI_DEV_FB_USED * on(cluster, namespace , pod) group_left(node) (node_namespace_pod:kube_pod_info:{$filter}) * 1024 * 1024`,
	"node_gpu_memory_available":   `DCGM_FI_DEV_FB_FREE* on(cluster, namespace , pod) group_left(node) (node_namespace_pod:kube_pod_info:{$filter}) * 1024 * 1024`,
	"node_gpu_memory_total":       `sum((DCGM_FI_DEV_FB_FREE + DCGM_FI_DEV_FB_USED) * on(cluster, pod,namespace) group_left(node) node_namespace_pod:kube_pod_info:{$filter}) by(node) * 1024 * 1024`,
	"node_gpu_temp":               `round(DCGM_FI_DEV_GPU_TEMP* on(cluster, namespace , pod) group_left(node) (node_namespace_pod:kube_pod_info:{$filter}), 0.001)`,
	"node_gpu_power_usage":        `round(DCGM_FI_DEV_POWER_USAGE* on(cluster, namespace , pod) group_left(node) (node_namespace_pod:kube_pod_info:{$filter}), 0.001)`,

	"node_device_size_usage":       `sum by(cluster, device, node, host_ip, role) (node_filesystem_size_bytes{device!~"/dev/loop\\d+",device=~"/dev/.*",job="node-exporter"} * on(cluster, namespace, pod) group_left(node, host_ip, role) node_namespace_pod:kube_pod_info:{$filter}) - sum by(cluster, device, node, host_ip, role) (node_filesystem_avail_bytes{device!~"/dev/loop\\d+",device=~"/dev/.*",job="node-exporter"} * on(cluster, namespace, pod) group_left(node, host_ip, role) node_namespace_pod:kube_pod_info:{$filter})`,
	"node_device_size_utilisation": `1 - sum by(cluster, device, node, host_ip, role) (node_filesystem_avail_bytes{device!~"/dev/loop\\d+",device=~"/dev/.*",job="node-exporter"} * on(cluster, namespace, pod) group_left(node, host_ip, role) node_namespace_pod:kube_pod_info:{$filter}) / sum by(device, node, host_ip, role) (node_filesystem_size_bytes{device!~"/dev/loop\\d+",device=~"/dev/.*",job="node-exporter"} * on(cluster, namespace, pod) group_left(node, host_ip, role) node_namespace_pod:kube_pod_info:{$filter})`,

	// workspace
	"workspace_cpu_usage":                  `round(sum by (cluster, workspace) (namespace:container_cpu_usage_seconds_total:sum_rate{namespace!="", $filter}), 0.001)`,
	"workspace_memory_usage":               `sum by (cluster, workspace) (namespace:container_memory_usage_bytes:sum{namespace!="", $filter})`,
	"workspace_memory_usage_wo_cache":      `sum by (cluster, workspace) (namespace:container_memory_usage_bytes_wo_cache:sum{namespace!="", $filter})`,
	"workspace_net_bytes_transmitted":      `sum by (cluster, workspace) (sum by (cluster, namespace) (irate(container_network_transmit_bytes_total{namespace!="", pod!="", interface!~"^(cali.+|tunl.+|dummy.+|kube.+|flannel.+|cni.+|docker.+|veth.+|lo.*)", job="kubelet"}[5m])) * on (cluster, namespace) group_left(workspace) kube_namespace_labels{$filter}) or on(cluster, workspace) max by(workspace) (kube_namespace_labels{$filter} * 0)`,
	"workspace_net_bytes_received":         `sum by (cluster, workspace) (sum by (cluster, namespace) (irate(container_network_receive_bytes_total{namespace!="", pod!="", interface!~"^(cali.+|tunl.+|dummy.+|kube.+|flannel.+|cni.+|docker.+|veth.+|lo.*)", job="kubelet"}[5m])) * on (cluster, namespace) group_left(workspace) kube_namespace_labels{$filter}) or on(cluster, workspace) max by(workspace) (kube_namespace_labels{$filter} * 0)`,
	"workspace_pod_count":                  `sum by (cluster, workspace) (kube_pod_status_phase{phase!~"Failed|Succeeded", namespace!=""} * on (cluster,namespace) group_left(workspace)(kube_namespace_labels{$filter})) or on(workspace) max by(workspace) (kube_namespace_labels{$filter} * 0)`,
	"workspace_pod_running_count":          `sum by (cluster, workspace) (kube_pod_status_phase{phase="Running", namespace!=""} * on (cluster, namespace) group_left(workspace)(kube_namespace_labels{$filter})) or on(workspace) max by(workspace) (kube_namespace_labels{$filter} * 0)`,
	"workspace_pod_succeeded_count":        `sum by (cluster, workspace) (kube_pod_status_phase{phase="Succeeded", namespace!=""} * on (cluster, namespace) group_left(workspace)(kube_namespace_labels{$filter})) or on(workspace) max by(workspace) (kube_namespace_labels{$filter} * 0)`,
	"workspace_pod_abnormal_count":         `count by (cluster, workspace) ((kube_pod_info{node!=""} unless on (cluster, pod, namespace) (kube_pod_status_phase{job="kube-state-metrics", phase="Succeeded"}>0) unless on (cluster, pod, namespace) ((kube_pod_status_ready{job="kube-state-metrics", condition="true"}>0) and on (cluster, pod, namespace) (kube_pod_status_phase{job="kube-state-metrics", phase="Running"}>0)) unless on (cluster, pod, namespace) (kube_pod_container_status_waiting_reason{job="kube-state-metrics", reason="ContainerCreating"}>0)) * on (cluster, namespace) group_left(workspace)(kube_namespace_labels{$filter}))`,
	"workspace_ingresses_extensions_count": `sum by (cluster, workspace) (kube_ingress_labels{namespace!=""} * on (cluster, namespace) group_left(workspace)(kube_namespace_labels{$filter}))`,
	"workspace_cronjob_count":              `sum by (cluster, workspace) (kube_cronjob_labels{namespace!=""} * on (cluster, namespace) group_left(workspace)(kube_namespace_labels{$filter}))`,
	"workspace_pvc_count":                  `sum by (cluster, workspace) (kube_persistentvolumeclaim_info{namespace!=""} * on (cluster, namespace) group_left(workspace)(kube_namespace_labels{$filter}))`,
	"workspace_daemonset_count":            `sum by (cluster, workspace) (kube_daemonset_labels{namespace!=""} * on (cluster, namespace) group_left(workspace)(kube_namespace_labels{$filter}))`,
	"workspace_deployment_count":           `sum by (cluster, workspace) (kube_deployment_labels{namespace!=""} * on (cluster, namespace) group_left(workspace)(kube_namespace_labels{$filter}))`,
	"workspace_endpoint_count":             `sum by (cluster, workspace) (kube_endpoint_labels{namespace!=""} * on (cluster, namespace) group_left(workspace)(kube_namespace_labels{$filter}))`,
	"workspace_hpa_count":                  `sum by (cluster, workspace) (kube_horizontalpodautoscaler_labels{namespace!=""} * on (cluster, namespace) group_left(workspace)(kube_namespace_labels{$filter}))`,
	"workspace_job_count":                  `sum by (cluster, workspace) (kube_job_labels{namespace!=""} * on (cluster, namespace) group_left(workspace)(kube_namespace_labels{$filter}))`,
	"workspace_statefulset_count":          `sum by (cluster, workspace) (kube_statefulset_labels{namespace!=""} * on (cluster, namespace) group_left(workspace)(kube_namespace_labels{$filter}))`,
	"workspace_replicaset_count":           `sum by (cluster, workspace) (kube_replicaset_labels{namespace!=""} * on (cluster, namespace) group_left(workspace)(kube_namespace_labels{$filter}))`,
	"workspace_service_count":              `sum by (cluster, workspace) (kube_service_info{namespace!=""} * on (cluster, namespace) group_left(workspace)(kube_namespace_labels{$filter}))`,
	"workspace_secret_count":               `sum by (cluster, workspace) (kube_secret_info{namespace!=""} * on (cluster, namespace) group_left(workspace)(kube_namespace_labels{$filter}))`,
	"workspace_pod_abnormal_ratio":         `count by (cluster, workspace) ((kube_pod_info{node!=""} unless on (pod, namespace) (kube_pod_status_phase{job="kube-state-metrics", phase="Succeeded"}>0) unless on (cluster, pod, namespace) ((kube_pod_status_ready{job="kube-state-metrics", condition="true"}>0) and on (cluster, pod, namespace) (kube_pod_status_phase{job="kube-state-metrics", phase="Running"}>0)) unless on (cluster, pod, namespace) (kube_pod_container_status_waiting_reason{job="kube-state-metrics", reason="ContainerCreating"}>0)) * on (cluster, namespace) group_left(workspace) kube_namespace_labels{$filter}) / sum by (workspace) (kube_pod_status_phase{phase!="Succeeded", namespace!=""} * on (cluster, namespace) group_left(workspace)(kube_namespace_labels{$filter}))`,
	"workspace_gpu_usage":                  `round((sum by(workspace) (label_replace(DCGM_FI_PROF_GR_ENGINE_ACTIVE{exported_namespace!=""},"namespace","$1","exported_namespace","(.+)") * on(cluster, namespace) group_left(workspace) (kube_namespace_labels{$filter})) or  sum by(workspace) (label_replace(DCGM_FI_DEV_GPU_UTIL{exported_namespace!=""},"namespace","$1","exported_namespace","(.+)") * on(cluster, namespace) group_left(workspace) (kube_namespace_labels{$filter})) ) / 100,0.001)`,
	"workspace_gpu_memory_usage":           `(sum by (workspace) (label_replace(DCGM_FI_DEV_FB_USED{exported_namespace!=""},"namespace","$1","exported_namespace","(.+)") * on(cluster, namespace) group_left(workspace) (kube_namespace_labels{$filter}))) * 1024 * 1024`,

	// namespace
	"namespace_cpu_usage":                        `round(namespace:container_cpu_usage_seconds_total:sum_rate{namespace!="", $filter}, 0.001)`,
	"namespace_cpu_used_requests_utilisation":    `round(namespace:container_cpu_usage_seconds_total:sum_rate{namespace!="", $filter} / on (cluster, namespace) sum by(cluster, namespace)(kube_pod_container_resource_requests{resource="cpu", $filter}), 0.001)`,
	"namespace_cpu_used_limits_utilisation":      `round(namespace:container_cpu_usage_seconds_total:sum_rate{namespace!="", $filter} / on (cluster, namespace) sum by(cluster, namespace)(kube_pod_container_resource_limits{resource="cpu", $filter}), 0.001)`,
	"namespace_memory_usage":                     `namespace:container_memory_usage_bytes:sum{namespace!="", $filter}`,
	"namespace_memory_usage_wo_cache":            `namespace:container_memory_usage_bytes_wo_cache:sum{namespace!="", $filter}`,
	"namespace_memory_used_requests_utilisation": `round(namespace:container_memory_usage_bytes_wo_cache:sum{namespace!="", $filter} / on (cluster, namespace) sum by(cluster, namespace)(kube_pod_container_resource_requests{resource="memory", $filter}),0.001)`,
	"namespace_memory_used_limits_utilisation":   `round(namespace:container_memory_usage_bytes_wo_cache:sum{namespace!="", $filter} / on (cluster, namespace) sum by(cluster, namespace)(kube_pod_container_resource_limitss{resource="memory", $filter}),0.001)`,
	"namespace_net_bytes_transmitted":            `sum by (cluster, namespace) (irate(container_network_transmit_bytes_total{namespace!="", pod!="", interface!~"^(cali.+|tunl.+|dummy.+|kube.+|flannel.+|cni.+|docker.+|veth.+|lo.*)", job="kubelet"}[5m]) * on (cluster, namespace) group_left(workspace) kube_namespace_labels{$filter}) or on(cluster, namespace) max by(cluster, namespace) (kube_namespace_labels{$filter} * 0)`,
	"namespace_net_bytes_received":               `sum by (cluster, namespace) (irate(container_network_receive_bytes_total{namespace!="", pod!="", interface!~"^(cali.+|tunl.+|dummy.+|kube.+|flannel.+|cni.+|docker.+|veth.+|lo.*)", job="kubelet"}[5m]) * on (cluster, namespace) group_left(workspace) kube_namespace_labels{$filter}) or on(cluster, namespace) max by(cluster, namespace) (kube_namespace_labels{$filter} * 0)`,
	"namespace_pod_count":                        `sum by (cluster, namespace) (kube_pod_status_phase{phase!~"Failed|Succeeded", namespace!=""} * on (cluster, namespace) group_left(workspace) kube_namespace_labels{$filter}) or on(namespace) max by(cluster, namespace) (kube_namespace_labels{$filter} * 0)`,
	"namespace_pod_running_count":                `sum by (cluster, namespace) (kube_pod_status_phase{phase="Running", namespace!=""} * on (cluster, namespace) group_left(workspace) kube_namespace_labels{$filter}) or on(cluster, namespace) max by(cluster, namespace) (kube_namespace_labels{$filter} * 0)`,
	"namespace_pod_succeeded_count":              `sum by (cluster, namespace) (kube_pod_status_phase{phase="Succeeded", namespace!=""} * on (cluster, namespace) group_left(workspace) kube_namespace_labels{$filter}) or on(cluster, namespace) max by(cluster, namespace) (kube_namespace_labels{$filter} * 0)`,
	"namespace_pod_abnormal_count":               `namespace:pod_abnormal:count{namespace!="", $filter}`,
	"namespace_pod_abnormal_ratio":               `namespace:pod_abnormal:ratio{namespace!="", $filter}`,
	"namespace_memory_limit_hard":                `min by (cluster, namespace) (kube_resourcequota{resourcequota!="quota", type="hard", namespace!="", resource="limits.memory"} * on (cluster, namespace) group_left(workspace) kube_namespace_labels{$filter})`,
	"namespace_cpu_limit_hard":                   `min by (cluster, namespace) (kube_resourcequota{resourcequota!="quota", type="hard", namespace!="", resource="limits.cpu"} * on (cluster, namespace) group_left(workspace) kube_namespace_labels{$filter})`,
	"namespace_pod_count_hard":                   `min by (cluster, namespace) (kube_resourcequota{resourcequota!="quota", type="hard", namespace!="", resource="count/pods"} * on (cluster, namespace) group_left(workspace) kube_namespace_labels{$filter})`,
	"namespace_pvc_bytes_used":                   `sum by (cluster, namespace) (kubelet_volume_stats_used_bytes * on (cluster, namespace) group_left(workspace) kube_namespace_labels{$filter})`,
	"namespace_pvc_bytes_utilisation":            `avg by (cluster, namespace) (kubelet_volume_stats_used_bytes / kubelet_volume_stats_capacity_bytes on (cluster, namespace) group_left(workspace) kube_namespace_labels{$filter})`,
	"namespace_cronjob_count":                    `sum by (cluster, namespace) (kube_cronjob_labels{namespace!=""} * on (cluster, namespace) group_left(workspace) kube_namespace_labels{$filter})`,
	"namespace_pvc_count":                        `sum by (cluster, namespace) (kube_persistentvolumeclaim_info{namespace!=""} * on (cluster, namespace) group_left(workspace) kube_namespace_labels{$filter})`,
	"namespace_daemonset_count":                  `sum by (cluster, namespace) (kube_daemonset_labels{namespace!=""} * on (cluster, namespace) group_left(workspace) kube_namespace_labels{$filter})`,
	"namespace_deployment_count":                 `sum by (cluster, namespace) (kube_deployment_labels{namespace!=""} * on (cluster, namespace) group_left(workspace) kube_namespace_labels{$filter})`,
	"namespace_endpoint_count":                   `sum by (cluster, namespace) (kube_endpoint_labels{namespace!=""} * on (cluster, namespace) group_left(workspace) kube_namespace_labels{$filter})`,
	"namespace_hpa_count":                        `sum by (cluster, namespace) (kube_horizontalpodautoscaler_labels{namespace!=""} * on (cluster, namespace) group_left(workspace) kube_namespace_labels{$filter})`,
	"namespace_job_count":                        `sum by (cluster, namespace) (kube_job_labels{namespace!=""} * on (cluster, namespace) group_left(workspace) kube_namespace_labels{$filter})`,
	"namespace_statefulset_count":                `sum by (cluster, namespace) (kube_statefulset_labels{namespace!=""} * on (cluster, namespace) group_left(workspace) kube_namespace_labels{$filter})`,
	"namespace_replicaset_count":                 `sum by (cluster, namespace) (kube_replicaset_labels{namespace!=""} * on (cluster, namespace) group_left(workspace) kube_namespace_labels{$filter})`,
	"namespace_service_count":                    `sum by (cluster, namespace) (kube_service_info{namespace!=""} * on (cluster, namespace) group_left(workspace) kube_namespace_labels{$filter})`,
	"namespace_secret_count":                     `sum by (cluster, namespace) (kube_secret_info{namespace!=""} * on (cluster, namespace) group_left(workspace) kube_namespace_labels{$filter})`,
	"namespace_configmap_count":                  `sum by (cluster, namespace) (kube_configmap_info{namespace!=""} * on (cluster, namespace) group_left(workspace) kube_namespace_labels{$filter})`,
	"namespace_ingresses_extensions_count":       `sum by (cluster, namespace) (kube_ingress_labels{namespace!=""} * on (cluster, namespace) group_left(workspace) kube_namespace_labels{$filter})`,
	"namespace_s2ibuilder_count":                 `sum by (cluster, namespace) (s2i_s2ibuilder_created{namespace!=""} * on (cluster, namespace) group_left(workspace) kube_namespace_labels{$filter})`,
	"namespace_gpu_limit_hard":                   `sum by (cluster, namespace) (kube_resourcequota{resource="requests.nvidia.com/gpu", type="hard", namespace!=""} * on (clustermnamespace) group_left(workspace) kube_namespace_labels{$filter})`,
	"namespace_gpu_usage":                        `round((sum by(cluster, namespace) (label_replace(DCGM_FI_PROF_GR_ENGINE_ACTIVE{exported_namespace!=""},"namespace","$1","exported_namespace","(.+)") * on(cluster, namespace) group_left(workspace) (kube_namespace_labels{$filter})) or  sum by(namespace) (label_replace(DCGM_FI_DEV_GPU_UTIL{exported_namespace!=""},"namespace","$1","exported_namespace","(.+)") * on(cluster,namespace) group_left( workspace) (kube_namespace_labels{$filter})) ) / 100,0.001)`,
	"namespace_gpu_memory_usage":                 `(sum by (cluster, namespace) (label_replace(DCGM_FI_DEV_FB_USED{exported_namespace!=""},"namespace","$1","exported_namespace","(.+)") * on(cluster, namespace) group_left(workspace) (kube_namespace_labels{$filter}))) * 1024 * 1024`,

	// ingress
	"ingress_request_count":                 `round(sum by(cluster)(increase(nginx_ingress_controller_requests{$filter}[$duration])))`,
	"ingress_request_4xx_count":             `round(sum by(cluster)(increase(nginx_ingress_controller_requests{$filter,status=~"[4].*"}[$duration])))`,
	"ingress_request_5xx_count":             `round(sum by(cluster)(increase(nginx_ingress_controller_requests{$filter,status=~"[5].*"}[$duration])))`,
	"ingress_active_connections":            `sum by(cluster)(avg_over_time(nginx_ingress_controller_nginx_process_connections{$filter,state="active"}[$duration]))`,
	"ingress_success_rate":                  `sum by(cluster)(rate(nginx_ingress_controller_requests{$filter,status!~"[4-5].*"}[$duration])) / sum(rate(nginx_ingress_controller_requests{$filter}[$duration]))`,
	"ingress_request_duration_average":      `sum_over_time(nginx_ingress_controller_request_duration_seconds_sum{$filter}[$duration])/sum_over_time(nginx_ingress_controller_request_duration_seconds_count{$filter}[$duration])`,
	"ingress_request_duration_50percentage": `histogram_quantile(0.50, sum by (cluster,le) (rate(nginx_ingress_controller_request_duration_seconds_bucket{$filter}[$duration])))`,
	"ingress_request_duration_95percentage": `histogram_quantile(0.95, sum by (cluster,le) (rate(nginx_ingress_controller_request_duration_seconds_bucket{$filter}[$duration])))`,
	"ingress_request_duration_99percentage": `histogram_quantile(0.99, sum by (cluster,le) (rate(nginx_ingress_controller_request_duration_seconds_bucket{$filter}[$duration])))`,
	"ingress_request_volume":                `round(sum by(cluster)(irate(nginx_ingress_controller_requests{$filter}[$duration])), 0.001)`,
	"ingress_request_volume_by_ingress":     `round(sum by(cluster, ingress)(irate(nginx_ingress_controller_requests{$filter}[$duration])), 0.001)`,
	"ingress_request_network_sent":          `sum by(cluster)(irate(nginx_ingress_controller_response_size_sum{$filter}[$duration]))`,
	"ingress_request_network_received":      `sum by(cluster)(irate(nginx_ingress_controller_request_size_sum{$filter}[$duration]))`,
	"ingress_request_memory_bytes":          `avg by(cluster)(nginx_ingress_controller_nginx_process_resident_memory_bytes{$filter})`,
	"ingress_request_cpu_usage":             `avg by(cluster)(rate(nginx_ingress_controller_nginx_process_cpu_seconds_total{$filter}[5m]))`,

	// workload
	"workload_cpu_usage":                              `round(namespace:workload_cpu_usage:sum{$filter}, 0.001)`,
	"workload_memory_usage":                           `namespace:workload_memory_usage:sum{$filter}`,
	"workload_memory_usage_wo_cache":                  `namespace:workload_memory_usage_wo_cache:sum{$filter}`,
	"workload_net_bytes_transmitted":                  `namespace:workload_net_bytes_transmitted:sum_irate{$filter}`,
	"workload_net_bytes_received":                     `namespace:workload_net_bytes_received:sum_irate{$filter}`,
	"workload_deployment_replica":                     `label_join(sum (label_join(label_replace(kube_deployment_spec_replicas{$filter}, "owner_kind", "Deployment", "", ""), "workload", "", "deployment")) by (namespace, owner_kind, workload), "workload", ":", "owner_kind", "workload")`,
	"workload_deployment_replica_available":           `label_join(sum (label_join(label_replace(kube_deployment_status_replicas_available{$filter}, "owner_kind", "Deployment", "", ""), "workload", "", "deployment")) by (namespace, owner_kind, workload), "workload", ":", "owner_kind", "workload")`,
	"workload_statefulset_replica":                    `label_join(sum (label_join(label_replace(kube_statefulset_replicas{$filter}, "owner_kind", "StatefulSet", "", ""), "workload", "", "statefulset")) by (namespace, owner_kind, workload), "workload", ":", "owner_kind", "workload")`,
	"workload_statefulset_replica_available":          `label_join(sum (label_join(label_replace(kube_statefulset_status_replicas_current{$filter}, "owner_kind", "StatefulSet", "", ""), "workload", "", "statefulset")) by (namespace, owner_kind, workload), "workload", ":", "owner_kind", "workload")`,
	"workload_daemonset_replica":                      `label_join(sum (label_join(label_replace(kube_daemonset_status_desired_number_scheduled{$filter}, "owner_kind", "DaemonSet", "", ""), "workload", "", "daemonset")) by (namespace, owner_kind, workload), "workload", ":", "owner_kind", "workload")`,
	"workload_daemonset_replica_available":            `label_join(sum (label_join(label_replace(kube_daemonset_status_number_available{$filter}, "owner_kind", "DaemonSet", "", ""), "workload", "", "daemonset")) by (namespace, owner_kind, workload), "workload", ":", "owner_kind", "workload")`,
	"workload_deployment_unavailable_replicas_ratio":  `namespace:deployment_unavailable_replicas:ratio{$filter}`,
	"workload_daemonset_unavailable_replicas_ratio":   `namespace:daemonset_unavailable_replicas:ratio{$filter}`,
	"workload_statefulset_unavailable_replicas_ratio": `namespace:statefulset_unavailable_replicas:ratio{$filter}`,
	"workload_gpu_usage":                              `round((sum by(cluster, namespace, owner_kind, workload) ((label_replace(label_replace(DCGM_FI_PROF_GR_ENGINE_ACTIVE{exported_namespace!=""},"namespace","$1","exported_namespace","(.+)"),"pod","$1","exported_pod","(.+)")) * on(cluster, pod, namespace) group_left(workload, owner_kind) (label_join(label_replace(kube_pod_owner{owner_name="<none>",$filetr},"owner_name","$1","pod","(.+)") or kube_pod_owner{owner_name!="<none>",}, "workload",":","owner_kind","owner_name")) )*  sum by(cluster, namespace, owner_kind, workload) (label_join(label_replace(kube_pod_owner{owner_name="<none>",$filetr},"owner_name","$1","pod","(.+)") or kube_pod_owner{owner_name!="<none>",}, "workload",":","owner_kind","owner_name")))/100, 0.001) or round((sum by(cluster, namespace, owner_kind, workload) ((label_replace(label_replace(DCGM_FI_DEV_GPU_UTIL{exported_namespace!=""},"namespace","$1","exported_namespace","(.+)"),"pod","$1","exported_pod","(.+)")) * on(cluster, pod, namespace) group_left(workload, owner_kind) (label_join(label_replace(kube_pod_owner{owner_name="<none>",$filetr},"owner_name","$1","pod","(.+)") or kube_pod_owner{owner_name!="<none>",}, "workload",":","owner_kind","owner_name")) )*  sum by(cluster, namespace, owner_kind, workload) (label_join(label_replace(kube_pod_owner{owner_name="<none>",$filetr},"owner_name","$1","pod","(.+)") or kube_pod_owner{owner_name!="<none>",}, "workload",":","owner_kind","owner_name")))/100, 0.001)`,
	"workload_gpu_memory_usage":                       `(sum by(cluster, namespace, owner_kind, workload) ((label_replace(label_replace(DCGM_FI_DEV_FB_USED{exported_namespace!=""},"namespace","$1","exported_namespace","(.+)"),"pod","$1","exported_pod","(.+)")) * on(cluster, pod, namespace) group_left(workload, owner_kind) (label_join(label_replace(kube_pod_owner{owner_name="<none>",$filetr},"owner_name","$1","pod","(.+)") or kube_pod_owner{owner_name!="<none>",$filetr}, "workload",":","owner_kind","owner_name")))) * 1024 * 1024`,

	// pod
	"pod_cpu_usage":                        `round(sum by (cluster,namespace, pod) (irate(container_cpu_usage_seconds_total{job="kubelet", pod!="", image!=""}[5m])) * on (cluster, namespace, pod) group_left (qos,owner_kind,owner_name,node) qos_owner_node:kube_pod_info:{$filter},0.001)`,
	"pod_cpu_used_requests_utilisation":    `round(sum by (cluster,namespace, pod) (irate(container_cpu_usage_seconds_total{job="kubelet", pod!="", image!="",$filter}[5m])) / sum by (cluster,namespace,pod)(kube_pod_container_resource_requests{resource="cpu",$filter}),0.001)* on (cluster, namespace, pod) group_left (qos,owner_kind,owner_name,node) qos_owner_node:kube_pod_info:`,
	"pod_cpu_used_limits_utilisation":      `round(sum by (cluster,namespace, pod) (irate(container_cpu_usage_seconds_total{job="kubelet", pod!="", image!="",$filter}[5m])) / sum by (cluster,namespace,pod)(kube_pod_container_resource_limits{resource="cpu",$filter}),0.001)* on (cluster, namespace, pod) group_left (qos,owner_kind,owner_name,node) qos_owner_node:kube_pod_info:`,
	"pod_memory_usage":                     `sum by (cluster, namespace, pod) (container_memory_usage_bytes{job="kubelet", pod!="", image!=""}) * on (cluster,namespace, pod) group_left (qos,owner_kind,owner_name,node) qos_owner_node:kube_pod_info:{$filter}`,
	"pod_memory_usage_wo_cache":            `sum by (cluster, namespace, pod) (container_memory_working_set_bytes{job="kubelet", pod!="", image!=""})  * on (cluster, namespace, pod) group_left (qos,owner_kind,owner_name,node) qos_owner_node:kube_pod_info:{$filter}`,
	"pod_memory_used_requests_utilisation": `round(sum by (cluster, namespace, pod) (container_memory_working_set_bytes{job="kubelet", pod!="", image!="",$filter}) / sum by (cluster, namespace ,pod)(kube_pod_container_resource_requests{resource="memory",$filter}), 0.001)* on (cluster, namespace, pod) group_left (qos,owner_kind,owner_name,node) qos_owner_node:kube_pod_info:`,
	"pod_memory_used_limits_utilisation":   `round(sum by (cluster, namespace, pod) (container_memory_working_set_bytes{job="kubelet", pod!="", image!="",$filter}) / sum by (cluster, namespace, pod)(kube_pod_container_resource_limits{resource="memory",$filter}), 0.001)* on (cluster, namespace, pod) group_left (qos,owner_kind,owner_name,node) qos_owner_node:kube_pod_info:`,
	"pod_net_bytes_transmitted":            `round(sum by (cluster, namespace, pod) (irate(container_network_transmit_bytes_total{pod!="", interface!~"^(cali.+|tunl.+|dummy.+|kube.+|flannel.+|cni.+|docker.+|veth.+|lo.*)", job="kubelet"}[5m])) * on (cluster, namespace, pod) group_left (qos,owner_kind,owner_name,node) qos_owner_node:kube_pod_info:{$filter}, 0.001)`,
	"pod_net_bytes_received":               `round(sum by (cluster, namespace, pod) (irate(container_network_receive_bytes_total{pod!="", interface!~"^(cali.+|tunl.+|dummy.+|kube.+|flannel.+|cni.+|docker.+|veth.+|lo.*)", job="kubelet"}[5m])) * on (cluster, namespace, pod) group_left (qos,owner_kind,owner_name,node) qos_owner_node:kube_pod_info:{$filter}, 0.001)`,
	"pod_pvc_bytes_used":                   `max by (cluster, namespace, persistentvolumeclaim) (kubelet_volume_stats_used_bytes) * on (cluster, namespace, persistentvolumeclaim) group_left (storageclass) kube_persistentvolumeclaim_info{} * on (cluster, namespace, persistentvolumeclaim) group_left (pod) kube_pod_spec_volumes_persistentvolumeclaims_info{$filter}* on (cluster, namespace, pod) group_left (qos,owner_kind,owner_name,node) qos_owner_node:kube_pod_info:`,
	"pod_pvc_bytes_utilisation":            `max by (cluster, namespace, persistentvolumeclaim, pod) (kubelet_volume_stats_used_bytes / kubelet_volume_stats_capacity_bytes) * on (cluster,namespace, persistentvolumeclaim) group_left (storageclass) kube_persistentvolumeclaim_info{} * on (cluster,namespace, persistentvolumeclaim) group_left (pod) kube_pod_spec_volumes_persistentvolumeclaims_info{$filter}* on (cluster, namespace, pod) group_left (qos,owner_kind,owner_name,node) qos_owner_node:kube_pod_info:`,
	"pod_gpu_usage":                        `round(((sum by(cluster,namespace, pod,qos,owner_kind,owner_name,node) (label_replace(label_replace(DCGM_FI_PROF_GR_ENGINE_ACTIVE{exported_namespace!=""},"namespace","$1","exported_namespace","(.+)"),"pod","$1","exported_pod","(.+)") *on (cluster, namespace, pod) group_left (qos,owner_kind,owner_name,node) qos_owner_node:kube_pod_info:{$filter}))) /100, 0.001) or round(((sum by(cluster,namespace, pod,qos,owner_kind,owner_name,node) (label_replace(label_replace(DCGM_FI_DEV_GPU_UTIL{exported_namespace!=""},"namespace","$1","exported_namespace","(.+)"),"pod","$1","exported_pod","(.+)") *on (cluster, namespace, pod) group_left (qos,owner_kind,owner_name,node) qos_owner_node:kube_pod_info:{$filter}))) /100, 0.001)`,
	"pod_gpu_memory_usage":                 `(sum by(cluster, namespace, pod) (label_replace(label_replace(DCGM_FI_DEV_FB_USED{exported_namespace!=""},"namespace","$1","exported_namespace","(.+)"),"pod","$1","exported_pod","(.+)")) *  on (cluster, namespace, pod) group_left (qos,owner_kind,owner_name,node) qos_owner_node:kube_pod_info:{$filter}) * 1024 * 1024 `,

	// container
	"container_cpu_usage":             `round(sum by (cluster, namespace, pod, container) (irate(container_cpu_usage_seconds_total{job="kubelet", container!="POD", container!="", image!="", $filter}[5m])), 0.001)`,
	"container_memory_usage":          `sum by (cluster, namespace, pod, container) (container_memory_usage_bytes{job="kubelet", container!="POD", container!="", image!="", $filter})`,
	"container_memory_usage_wo_cache": `sum by (cluster, namespace, pod, container) (container_memory_working_set_bytes{job="kubelet", container!="POD", container!="", image!="", $filter})`,
	"container_processes_usage":       `sum by (cluster, namespace, pod, container) (container_processes{job="kubelet", container!="POD", container!="", image!="", $filter})`,
	"container_threads_usage":         `sum by (cluster, namespace, pod, container) (container_threads {job="kubelet", container!="POD", container!="", image!="", $filter})`,
	"container_gpu_usage":             `round((sum by(cluster, namespace, pod, container, device, gpu) (label_replace(label_replace(label_replace(DCGM_FI_PROF_GR_ENGINE_ACTIVE{},"namespace","$1","exported_namespace","(.+)"),"pod","$1","exported_pod","(.+)"),"container","$1","exported_container","(.+)"))*on (cluster, namespace, pod)group_left(qos) qos_owner_node:kube_pod_info:{$filter} or sum by(cluster,namespace, pod, container, device, gpu) (label_replace(label_replace(label_replace(DCGM_FI_DEV_GPU_UTIL{},"namespace","$1","exported_namespace","(.+)"),"pod","$1","exported_pod","(.+)"),"container","$1","exported_container","(.+)"))*on (cluster, namespace, pod)group_left(qos) qos_owner_node:kube_pod_info:{$filter} )/100 , 0.001) `,
	"container_gpu_memory_usage":      `(sum by(cluster,namespace, pod, container) (label_replace(label_replace(label_replace(DCGM_FI_DEV_FB_USED{},"namespace","$1","exported_namespace","(.+)"),"pod","$1","exported_pod","(.+)"),"container","$1","exported_container","(.+)") * on (cluster, namespace, pod) group_left(qos) qos_owner_node:kube_pod_info:{$filter})) * 1024 * 1024 `,

	// pvc
	"pvc_inodes_available":   `max by (cluster, namespace, persistentvolumeclaim) (kubelet_volume_stats_inodes_free) * on (cluster, namespace, persistentvolumeclaim) group_left (storageclass) kube_persistentvolumeclaim_info{$filter}`,
	"pvc_inodes_used":        `max by (cluster, namespace, persistentvolumeclaim) (kubelet_volume_stats_inodes_used) * on (cluster, namespace, persistentvolumeclaim) group_left (storageclass) kube_persistentvolumeclaim_info{$filter}`,
	"pvc_inodes_total":       `max by (cluster, namespace, persistentvolumeclaim) (kubelet_volume_stats_inodes) * on (cluster, namespace, persistentvolumeclaim) group_left (storageclass) kube_persistentvolumeclaim_info{$filter}`,
	"pvc_inodes_utilisation": `max by (cluster, namespace, persistentvolumeclaim) (kubelet_volume_stats_inodes_used / kubelet_volume_stats_inodes) * on (cluster, namespace, persistentvolumeclaim) group_left (storageclass) kube_persistentvolumeclaim_info{$filter}`,
	"pvc_bytes_available":    `max by (cluster, namespace, persistentvolumeclaim) (kubelet_volume_stats_available_bytes) * on (cluster, namespace, persistentvolumeclaim) group_left (storageclass) kube_persistentvolumeclaim_info{$filter}`,
	"pvc_bytes_used":         `max by (cluster, namespace, persistentvolumeclaim) (kubelet_volume_stats_used_bytes) * on (cluster, namespace, persistentvolumeclaim) group_left (storageclass) kube_persistentvolumeclaim_info{$filter}`,
	"pvc_bytes_total":        `max by (cluster, namespace, persistentvolumeclaim) (kubelet_volume_stats_capacity_bytes) * on (cluster, namespace, persistentvolumeclaim) group_left (storageclass) kube_persistentvolumeclaim_info{$filter}`,
	"pvc_bytes_utilisation":  `max by (cluster, namespace, persistentvolumeclaim) (kubelet_volume_stats_used_bytes / kubelet_volume_stats_capacity_bytes) * on (cluster, namespace, persistentvolumeclaim) group_left (storageclass) kube_persistentvolumeclaim_info{$filter}`,

	// component
	"etcd_server_list":                           `label_replace(up{job="etcd", $filter}, "node_ip", "$1", "instance", "(.*):.*")`,
	"etcd_server_total":                          `count(up{job="etcd", $filter})`,
	"etcd_server_up_total":                       `etcd:up:sum{$filter}`,
	"etcd_server_has_leader":                     `label_replace(etcd_server_has_leader{$filter}, "node_ip", "$1", "instance", "(.*):.*")`,
	"etcd_server_is_leader":                      `label_replace(etcd_server_is_leader{$filter}, "node_ip", "$1", "instance", "(.*):.*")`,
	"etcd_server_leader_changes":                 `label_replace(etcd:etcd_server_leader_changes_seen:sum_changes{$filter}, "node_ip", "$1", "node", "(.*)")`,
	"etcd_server_proposals_failed_rate":          `avg by (cluster) (etcd:etcd_server_proposals_failed:sum_irate{$filter})`,
	"etcd_server_proposals_applied_rate":         `avg by (cluster) (etcd:etcd_server_proposals_applied:sum_irate{$filter})`,
	"etcd_server_proposals_committed_rate":       `avg by (cluster) (etcd:etcd_server_proposals_committed:sum_irate{$filter})`,
	"etcd_server_proposals_pending_count":        `avg by (cluster) (etcd:etcd_server_proposals_pending:sum{$filter})`,
	"etcd_mvcc_db_size":                          `avg by (cluster) (etcd:etcd_mvcc_db_total_size:sum{$filter})`,
	"etcd_network_client_grpc_received_bytes":    `sum by (cluster) (etcd:etcd_network_client_grpc_received_bytes:sum_irate{$filter})`,
	"etcd_network_client_grpc_sent_bytes":        `sum by (cluster) (etcd:etcd_network_client_grpc_sent_bytes:sum_irate{$filter})`,
	"etcd_grpc_call_rate":                        `sum by (cluster) (etcd:grpc_server_started:sum_irate{$filter})`,
	"etcd_grpc_call_failed_rate":                 `sum by (cluster) (etcd:grpc_server_handled:sum_irate{$filter})`,
	"etcd_grpc_server_msg_received_rate":         `sum by (cluster) (etcd:grpc_server_msg_received:sum_irate{$filter})`,
	"etcd_grpc_server_msg_sent_rate":             `sum by (cluster) (etcd:grpc_server_msg_sent:sum_irate{$filter})`,
	"etcd_disk_wal_fsync_duration":               `avg by (cluster) (etcd:etcd_disk_wal_fsync_duration:avg{$filter})`,
	"etcd_disk_wal_fsync_duration_quantile":      `avg by (cluster, quantile) (etcd:etcd_disk_wal_fsync_duration:histogram_quantile{$filter})`,
	"etcd_disk_backend_commit_duration":          `avg by (cluster) (etcd:etcd_disk_backend_commit_duration:avg{$filter})`,
	"etcd_disk_backend_commit_duration_quantile": `avg by (cluster, quantile)(etcd:etcd_disk_backend_commit_duration:histogram_quantile{$filter})`,

	"apiserver_up_sum":                    `apiserver:up:sum{$filter}`,
	"apiserver_request_rate":              `apiserver:apiserver_request_total:sum_irate{$filter}`,
	"apiserver_request_by_verb_rate":      `apiserver:apiserver_request_total:sum_verb_irate{$filter}`,
	"apiserver_request_latencies":         `apiserver:apiserver_request_duration:avg{$filter}`,
	"apiserver_request_by_verb_latencies": `apiserver:apiserver_request_duration:avg_by_verb{$filter}`,

	"scheduler_up_sum":                          `scheduler:up:sum{$filter}`,
	"scheduler_schedule_attempts":               `scheduler:scheduler_schedule_attempts:sum{$filter}`,
	"scheduler_schedule_attempt_rate":           `scheduler:scheduler_schedule_attempts:sum_rate{$filter}`,
	"scheduler_e2e_scheduling_latency":          `scheduler:scheduler_e2e_scheduling_duration:avg{$filter}`,
	"scheduler_e2e_scheduling_latency_quantile": `scheduler:scheduler_e2e_scheduling_duration:histogram_quantile{$filter}`,
}

var protectedMetrics = map[string]bool{
	"workspace_gpu_usage":        true,
	"workspace_gpu_memory_usage": true,
	"namespace_gpu_usage":        true,
	"namespace_gpu_memory_usage": true,
	"workload_gpu_usage":         true,
	"workload_gpu_memory_usage":  true,
	"pod_gpu_usage":              true,
	"pod_gpu_memory_usage":       true,
	"container_gpu_usage":        true,
	"container_gpu_memory_usage": true,
}

var wrappedQueryMetrics = map[string]bool{
	"container_gpu_usage":        true,
	"container_gpu_memory_usage": true,
}

func makeExpr(metric string, opts monitoring.QueryOptions) string {
	// Consider the "$1" in label_replace:
	// wrappedExpr converts `"$1"` to `$labelReplace`,
	// once completed, will convert back to `"$1"`.
	tmpl := promQLTemplates[metric]
	return templateExpr(metric, tmpl, opts)
}

func templateExpr(metric string, tmpl string, opts monitoring.QueryOptions) string {
	switch opts.Level {
	case monitoring.LevelPod:
		return makePodMetricExpr(tmpl, opts)
	case monitoring.LevelWorkload:
		return makeWorkloadMetricExpr(metric, tmpl, opts)
	case monitoring.LevelIngress:
		return makeIngressMetricExpr(tmpl, opts)
	default:
		return makeSampleMetricExpr(tmpl, opts)
	}
}

func makeSampleMetricExpr(tmpl string, o monitoring.QueryOptions) string {
	var selector []string

	if o.ClusterName != "" {
		selector = append(selector, fmt.Sprintf(`cluster="%s"`, o.ClusterName))
	} else if o.ClusterResourcesFilter != "" {
		selector = append(selector, fmt.Sprintf(`cluster=~"%s"`, o.ClusterResourcesFilter))
	}

	if o.NodeName != "" {
		selector = append(selector, fmt.Sprintf(`node="%s"`, o.NodeName))
	} else if o.NodeResourcesFilter != "" {
		selector = append(selector, fmt.Sprintf(`node=~"%s"`, o.NodeResourcesFilter))
	}

	if o.WorkspaceName != "" {
		selector = append(selector, fmt.Sprintf(`workspace="%s"`, o.WorkspaceName))
	}
	if o.WorkspaceResourcesFilter != "" {
		selector = append(selector, fmt.Sprintf(`workspace=~"%s"`, o.WorkspaceResourcesFilter))
	}

	if o.NamespaceName != "" {
		selector = append(selector, fmt.Sprintf(`namespace="%s"`, o.NamespaceName))
	} else if o.NamespaceResourcesFilter != "" {
		selector = append(selector, fmt.Sprintf(`namespace=~"%s"`, o.NamespaceResourcesFilter))
	}

	if o.PodName != "" {
		selector = append(selector, fmt.Sprintf(`pod="%s"`, o.PodName))
	} else if o.PodResourcesFilter != "" {
		selector = append(selector, fmt.Sprintf(`pod=~"%s"`, o.PodResourcesFilter))
	}

	if o.ContainerName != "" {
		selector = append(selector, fmt.Sprintf(`container="%s"`, o.ContainerName))
	} else if o.ContainerResourcesFilter != "" {
		selector = append(selector, fmt.Sprintf(`container=~"%s"`, o.ContainerResourcesFilter))
	}

	if o.IngressName != "" {
		selector = append(selector, fmt.Sprintf(`ingress="%s"`, o.IngressName))
	} else if o.IngressResourcesFilter != "" {
		selector = append(selector, fmt.Sprintf(`ingress=~"%s"`, o.IngressResourcesFilter))
	}

	if o.StorageClassName != "" {
		selector = append(selector, fmt.Sprintf(`storageclass="%s"`, o.StorageClassName))
	} else if o.StorageClassResourcesFilter != "" {
		selector = append(selector, fmt.Sprintf(`storageclass=~"%s"`, o.StorageClassResourcesFilter))
	}

	if o.PVCName != "" {
		selector = append(selector, fmt.Sprintf(`persistentvolumeclaim="%s"`, o.PVCName))
	} else if o.PVCResourcesFilter != "" {
		selector = append(selector, fmt.Sprintf(`persistentvolumeclaim=~"%s"`, o.PVCResourcesFilter))
	}

	return strings.Replace(tmpl, "$filter", strings.Join(selector, ","), -1)
}

func makeIngressMetricExpr(tmpl string, o monitoring.QueryOptions) string {
	var selector []string
	duration := "5m"

	// parse Range Vector Selectors metric{key=value}[duration]
	if o.Duration != nil {
		duration = o.Duration.String()
	}

	if o.ClusterName != "" {
		selector = append(selector, fmt.Sprintf(`cluster="%s"`, o.ClusterName))
	} else if o.ClusterResourcesFilter != "" {
		selector = append(selector, fmt.Sprintf(`cluster=~"%s"`, o.ClusterResourcesFilter))
	}

	// For monitoring ingress in the specific namespace
	// GET /namespaces/{namespace}/ingress/{ingress} or
	// GET /namespaces/{namespace}/ingress
	if o.NamespaceName != constants.KubeSphereNamespace {
		if o.IngressName != "" {
			selector = append(selector, fmt.Sprintf(`exported_namespace="%s", ingress="%s"`, o.NamespaceName, o.IngressName))
		} else {
			selector = append(selector, fmt.Sprintf(`exported_namespace="%s", ingress=~"%s"`, o.NamespaceName, o.IngressResourcesFilter))
		}
	} else {
		if o.IngressName != "" {
			selector = append(selector, fmt.Sprintf(`ingress="%s"`, o.IngressName))
		} else {
			selector = append(selector, fmt.Sprintf(`ingress=~"%s"`, o.IngressResourcesFilter))
		}
	}

	// job is a reqiuried filter
	// GET /namespaces/{namespace}/ingress?job=xxx&pod=xxx
	if o.Job != "" {
		selector = append(selector, fmt.Sprintf(`job="%s"`, o.Job))
		if o.PodName != "" {
			selector = append(selector, fmt.Sprintf(`controller_pod="%s"`, o.PodName))
		}
	}

	tmpl = strings.Replace(tmpl, "$duration", duration, -1)
	return strings.Replace(tmpl, "$filter", strings.Join(selector, ","), -1)
}

func makeWorkloadMetricExpr(metric, tmpl string, o monitoring.QueryOptions) string {
	var selector []string

	//GET /clusters/{cluster}/namespaces/{namespace}/workloads
	//GET /clusters/{cluster}/namespaces/{namespace}/workloads/{kind}
	switch o.WorkloadKind {
	case "deployment":
		o.WorkloadKind = Deployment
	case "statefulset":
		o.WorkloadKind = StatefulSet
	case "daemonset":
		o.WorkloadKind = DaemonSet
	default:
		o.WorkloadKind = ".*"
	}

	if o.ClusterName != "" {
		selector = append(selector, fmt.Sprintf(`cluster="%s"`, o.ClusterName))
	} else if o.ClusterResourcesFilter != "" {
		selector = append(selector, fmt.Sprintf(`cluster=~"%s"`, o.ClusterResourcesFilter))
	}

	if o.NamespaceName != "" {
		selector = append(selector, fmt.Sprintf(`namespace="%s"`, o.NamespaceName))
	} else if o.NamespaceResourcesFilter != "" {
		selector = append(selector, fmt.Sprintf(`namespace=~"%s"`, o.NamespaceResourcesFilter))
	}

	var workloadSelector string
	workloadSelector = fmt.Sprintf(`workload=~"%s:(%s)"`, o.WorkloadKind, o.WorkloadResourcesFilter)

	if strings.Contains(metric, "deployment") {
		workloadSelector = fmt.Sprintf(`deployment!="", deployment=~"%s"`, o.WorkloadResourcesFilter)
	}
	if strings.Contains(metric, "statefulset") {
		workloadSelector = fmt.Sprintf(`statefulset!="", statefulset=~"%s"`, o.WorkloadResourcesFilter)
	}
	if strings.Contains(metric, "daemonset") {
		workloadSelector = fmt.Sprintf(`daemonset!="", daemonset=~"%s"`, o.WorkloadResourcesFilter)
	}
	selector = append(selector, workloadSelector)
	return strings.Replace(tmpl, "$filter", strings.Join(selector, ","), -1)
}

func makePodMetricExpr(tmpl string, o monitoring.QueryOptions) string {
	var selector []string

	if o.ClusterName != "" {
		selector = append(selector, fmt.Sprintf(`cluster="%s"`, o.ClusterName))
	} else if o.ClusterResourcesFilter != "" {
		selector = append(selector, fmt.Sprintf(`cluster=~"%s"`, o.ClusterResourcesFilter))
	}

	if o.NodeName != "" {
		selector = append(selector, fmt.Sprintf(`node="%s"`, o.NodeName))
	} else if o.NodeResourcesFilter != "" {
		selector = append(selector, fmt.Sprintf(`node=~"%s"`, o.NodeResourcesFilter))
	}

	// GET /clusters/{cluster}/pods
	// GET /clusters/{cluster}/nodes/{node}/pods
	// namespaced_resources_filter like `<namespace>/<pod_name>|<namespace>/<pod_name>`
	if o.NamespacedNameResourcesFilter != "" {
		var namespaces, pods []string

		for _, np := range strings.Split(o.NamespacedNameResourcesFilter, "|") {
			if nparr := strings.SplitN(np, "/", 2); len(nparr) > 1 {
				namespaces = append(namespaces, nparr[0])
				pods = append(pods, nparr[1])
			} else {
				pods = append(pods, np)
			}
		}

		namespaces = append(namespaces, o.NamespaceResourcesFilter)
		o.NamespaceResourcesFilter = strings.Join(namespaces, "|")

		pods = append(pods, o.PodResourcesFilter)
		o.PodResourcesFilter = strings.Join(pods, "|")
	}

	if o.NamespaceName != "" {
		selector = append(selector, fmt.Sprintf(`namespace="%s"`, o.NamespaceName))
	} else if o.NamespaceResourcesFilter != "" {
		selector = append(selector, fmt.Sprintf(`namespace=~"%s"`, o.NamespaceResourcesFilter))
	}
	// For monitoriong pods of the specific workload
	// GET /namespaces/{namespace}/workloads/{kind}/{workload}/pods
	if o.WorkloadName != "" {
		switch o.WorkloadKind {
		case "deployment":
			selector = append(selector, fmt.Sprintf(`owner_kind="ReplicaSet", owner_name=~"^%s-[^-]{1,10}$"`, o.WorkloadName))
		case "statefulset":
			selector = append(selector, fmt.Sprintf(`owner_kind="StatefulSet", owner_name="%s"`, o.WorkloadName))
		case "daemonset":
			selector = append(selector, fmt.Sprintf(`owner_kind="DaemonSet", owner_name="%s"`, o.WorkloadName))
		}
	}
	if o.PodName != "" {
		selector = append(selector, fmt.Sprintf(`pod="%s"`, o.PodName))
	} else if o.PodResourcesFilter != "" {
		selector = append(selector, fmt.Sprintf(`pod=~"%s"`, o.PodResourcesFilter))
	}

	return strings.Replace(tmpl, "$filter", strings.Join(selector, ","), -1)
}

// wrappedExpr converts `"$1"`  to `$labelReplace`
func wrappedExpr(tmpl string) string {
	return strings.Replace(tmpl, "\"$1\"", "$labelReplace", -1)
}

// wrappedExpr converts `$labelReplace` back to `"$1"`
func unWrappedExpr(tmpl string) string {
	return strings.Replace(tmpl, "$labelReplace", "\"$1\"", -1)
}
