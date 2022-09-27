/*

 Copyright 2022 The KubeSphere Authors.

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

package v1alpha1

import (
	"net/http"

	"github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"

	"kubesphere.io/kubesphere/pkg/apiserver/runtime"
	"kubesphere.io/kubesphere/pkg/constants"
	"kubesphere.io/kubesphere/pkg/informers"
	model "kubesphere.io/kubesphere/pkg/models/observability/monitoring"
	"kubesphere.io/kubesphere/pkg/simple/client/observability/monitoring"
)

const (
	groupName = "monitoring.observability.kubesphere.io"
	respOK    = "ok"
)

var GroupVersion = schema.GroupVersion{Group: groupName, Version: "v1alpha1"}

func AddToContainer(c *restful.Container, k8sClient kubernetes.Interface, monitoringClient monitoring.Interface, factory informers.InformerFactory) error {
	ws := runtime.NewWebService(GroupVersion)

	h := NewHandler(k8sClient, monitoringClient, factory)

	ws.Route(ws.GET("/kubesphere").
		To(h.handleKubeSphereMetricsQuery).
		Doc("Get platform-level metric data for all clusters.").
		Metadata(restfulspec.KeyOpenAPITags, []string{constants.KubeSphereMetricsTag}).
		Writes(model.Metrics{}).
		Returns(http.StatusOK, respOK, model.Metrics{})).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/clusters").
		To(h.handleClusterMetricsQuery).
		Doc("Get cluster-level metric data for all clusters.").
		Param(ws.QueryParameter("metrics_filter", "The metric name filter consists of a regexp pattern. It specifies which metric data to return. For example, the following filter matches both cluster CPU usage and disk usage: `cluster_cpu_usage|cluster_disk_size_usage`. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("resources_filter", "The cluster filter consists of a regexp pattern. It specifies which cluster data to return. For example, the following filter matches both cluster host and member-a: `host|member-a`").DataType("string").Required(false)).
		Param(ws.QueryParameter("start", "Start time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1559347200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("end", "End time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1561939200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("step", "Time interval. Retrieve metric data at a fixed interval within the time range of start and end. It requires both **start** and **end** are provided. The format is [0-9]+[smhdwy]. Defaults to 10m (i.e. 10 min).").DataType("string").DefaultValue("10m").Required(false)).
		Param(ws.QueryParameter("time", "A timestamp in Unix time format. Retrieve metric data at a single point in time. Defaults to now. Time and the combination of start, end, step are mutually exclusive.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_metric", "Sort nodes by the specified metric. Not applicable if **start** and **end** are provided.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_type", "Sort order. One of asc, desc.").DefaultValue("desc.").DataType("string").Required(false)).
		Param(ws.QueryParameter("page", "The page number. This field paginates result data of each metric, then returns a specific page. For example, setting **page** to 2 returns the second page. It only applies to sorted metric data.").DataType("integer").Required(false)).
		Param(ws.QueryParameter("limit", "Page size, the maximum number of results in a single page. Defaults to 5.").DataType("integer").Required(false).DefaultValue("5")).
		Param(ws.QueryParameter("type", "The query type. This field can be set to 'rank' for cluster ranking query or '' for others. Defaults to ''.").DataType("string").Required(false).DefaultValue("")).
		Metadata(restfulspec.KeyOpenAPITags, []string{constants.ClusterMetricsTag}).
		Writes(model.Metrics{}).
		Returns(http.StatusOK, respOK, model.Metrics{})).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/clusters/{cluster}").
		To(h.handleClusterMetricsQuery).
		Doc("Get cluster-level metric data of the specific cluster.").
		Param(ws.PathParameter("cluster", "The name of cluster").DataType("string").Required(true)).
		Param(ws.QueryParameter("metrics_filter", "The metric name filter consists of a regexp pattern. It specifies which metric data to return. For example, the following filter matches both cluster CPU usage and disk usage: `cluster_cpu_usage|cluster_disk_size_usage`. View available metrics at [kubesphere.io](https://docs.kubesphere.io/advanced-v2.0/zh-CN/api-reference/monitoring-metrics/).").DataType("string").Required(false)).
		Param(ws.QueryParameter("start", "Start time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1559347200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("end", "End time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1561939200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("step", "Time interval. Retrieve metric data at a fixed interval within the time range of start and end. It requires both **start** and **end** are provided. The format is [0-9]+[smhdwy]. Defaults to 10m (i.e. 10 min).").DataType("string").DefaultValue("10m").Required(false)).
		Param(ws.QueryParameter("time", "A timestamp in Unix time format. Retrieve metric data at a single point in time. Defaults to now. Time and the combination of start, end, step are mutually exclusive.").DataType("string").Required(false)).
		Metadata(restfulspec.KeyOpenAPITags, []string{constants.ClusterMetricsTag}).
		Writes(model.Metrics{}).
		Returns(http.StatusOK, respOK, model.Metrics{})).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/nodes").
		To(h.handleNodeMetricsQuery).
		Doc("Get node-level metric data for all cluster's nodes.").
		Param(ws.QueryParameter("metrics_filter", "The metric name filter consists of a regexp pattern. It specifies which metric data to return. For example, the following filter matches both node CPU usage and disk usage: `node_cpu_usage|node_disk_size_usage`. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("resources_filter", "The node filter consists of a regexp pattern. It specifies which node data to return. For example, the following filter matches both node i-caojnter and i-cmu82ogj: `i-caojnter|i-cmu82ogj`.").DataType("string").Required(false)).
		Param(ws.QueryParameter("cluster_resources_filter", "The cluster filter consists of a regexp pattern. It specifies which cluster data to return. For example, the following filter matches both cluster host and member-a: `host|member-a`.").DataType("string").Required(false)).
		Param(ws.QueryParameter("start", "Start time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1559347200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("end", "End time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1561939200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("step", "Time interval. Retrieve metric data at a fixed interval within the time range of start and end. It requires both **start** and **end** are provided. The format is [0-9]+[smhdwy]. Defaults to 10m (i.e. 10 min).").DataType("string").DefaultValue("10m").Required(false)).
		Param(ws.QueryParameter("time", "A timestamp in Unix time format. Retrieve metric data at a single point in time. Defaults to now. Time and the combination of start, end, step are mutually exclusive.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_metric", "Sort nodes by the specified metric. Not applicable if **start** and **end** are provided.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_type", "Sort order. One of asc, desc.").DefaultValue("desc.").DataType("string").Required(false)).
		Param(ws.QueryParameter("page", "The page number. This field paginates result data of each metric, then returns a specific page. For example, setting **page** to 2 returns the second page. It only applies to sorted metric data.").DataType("integer").Required(false)).
		Param(ws.QueryParameter("limit", "Page size, the maximum number of results in a single page. Defaults to 5.").DataType("integer").Required(false).DefaultValue("5")).
		Param(ws.QueryParameter("type", "The query type. This field can be set to 'rank' for node ranking query or '' for others. Defaults to ''.").DataType("string").Required(false).DefaultValue("")).
		Metadata(restfulspec.KeyOpenAPITags, []string{constants.NodeMetricsTag}).
		Writes(model.Metrics{}).
		Returns(http.StatusOK, respOK, model.Metrics{})).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/clusters/{cluster}/nodes").
		To(h.handleNodeMetricsQuery).
		Doc("Get node-level metric data for all nodes of the specific cluster.").
		Param(ws.PathParameter("cluster", "The name of cluster").DataType("string").Required(true)).
		Param(ws.QueryParameter("metrics_filter", "The metric name filter consists of a regexp pattern. It specifies which metric data to return. For example, the following filter matches both node CPU usage and disk usage: `node_cpu_usage|node_disk_size_usage`. View available metrics at [kubesphere.io](https://docs.kubesphere.io/advanced-v2.0/zh-CN/api-reference/monitoring-metrics/).").DataType("string").Required(false)).
		Param(ws.QueryParameter("resources_filter", "The node filter consists of a regexp pattern. It specifies which node data to return. For example, the following filter matches both node i-caojnter and i-cmu82ogj: `i-caojnter|i-cmu82ogj`.").DataType("string").Required(false)).
		Param(ws.QueryParameter("start", "Start time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1559347200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("end", "End time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1561939200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("step", "Time interval. Retrieve metric data at a fixed interval within the time range of start and end. It requires both **start** and **end** are provided. The format is [0-9]+[smhdwy]. Defaults to 10m (i.e. 10 min).").DataType("string").DefaultValue("10m").Required(false)).
		Param(ws.QueryParameter("time", "A timestamp in Unix time format. Retrieve metric data at a single point in time. Defaults to now. Time and the combination of start, end, step are mutually exclusive.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_metric", "Sort nodes by the specified metric. Not applicable if **start** and **end** are provided.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_type", "Sort order. One of asc, desc.").DefaultValue("desc.").DataType("string").Required(false)).
		Param(ws.QueryParameter("page", "The page number. This field paginates result data of each metric, then returns a specific page. For example, setting **page** to 2 returns the second page. It only applies to sorted metric data.").DataType("integer").Required(false)).
		Param(ws.QueryParameter("limit", "Page size, the maximum number of results in a single page. Defaults to 5.").DataType("integer").Required(false).DefaultValue("5")).
		Param(ws.QueryParameter("type", "The query type. This field can be set to 'rank' for node ranking query or '' for others. Defaults to ''.").DataType("string").Required(false).DefaultValue("")).
		Metadata(restfulspec.KeyOpenAPITags, []string{constants.NodeMetricsTag}).
		Writes(model.Metrics{}).
		Returns(http.StatusOK, respOK, model.Metrics{})).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/clusters/{cluster}/nodes/{node}").
		To(h.handleNodeMetricsQuery).
		Doc("Get node-level metric data for the specific node of the specific cluster.").
		Param(ws.PathParameter("cluster", "The name of cluster").DataType("string").Required(true)).
		Param(ws.PathParameter("node", "Node name.").DataType("string").Required(true)).
		Param(ws.QueryParameter("metrics_filter", "The metric name filter consists of a regexp pattern. It specifies which metric data to return. For example, the following filter matches both node CPU usage and disk usage: `node_cpu_usage|node_disk_size_usage`. View available metrics at [kubesphere.io](https://docs.kubesphere.io/advanced-v2.0/zh-CN/api-reference/monitoring-metrics/).").DataType("string").Required(false)).
		Param(ws.QueryParameter("start", "Start time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1559347200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("end", "End time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1561939200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("step", "Time interval. Retrieve metric data at a fixed interval within the time range of start and end. It requires both **start** and **end** are provided. The format is [0-9]+[smhdwy]. Defaults to 10m (i.e. 10 min).").DataType("string").DefaultValue("10m").Required(false)).
		Param(ws.QueryParameter("time", "A timestamp in Unix time format. Retrieve metric data at a single point in time. Defaults to now. Time and the combination of start, end, step are mutually exclusive.").DataType("string").Required(false)).
		Metadata(restfulspec.KeyOpenAPITags, []string{constants.NodeMetricsTag}).
		Writes(model.Metrics{}).
		Returns(http.StatusOK, respOK, model.Metrics{})).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/workspaces").
		To(h.handleWorkspaceMetricsQuery).
		Doc("Get workspace-level metric data for all cluster's workspaces.").
		Param(ws.QueryParameter("metrics_filter", "The metric name filter consists of a regexp pattern. It specifies which metric data to return. For example, the following filter matches both workspace CPU usage and memory usage: `workspace_cpu_usage|workspace_memory_usage`. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("resources_filter", "The workspace filter consists of a regexp pattern. It specifies which workspace data to return.").DataType("string").Required(false)).
		Param(ws.QueryParameter("cluster_resources_filter", "The cluster filter consists of a regexp pattern. It specifies which cluster data to return. For example, the following filter matches both cluster host and member-a: `host|member-a`.").DataType("string").Required(false)).
		Param(ws.QueryParameter("start", "Start time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1559347200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("end", "End time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1561939200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("step", "Time interval. Retrieve metric data at a fixed interval within the time range of start and end. It requires both **start** and **end** are provided. The format is [0-9]+[smhdwy]. Defaults to 10m (i.e. 10 min).").DataType("string").DefaultValue("10m").Required(false)).
		Param(ws.QueryParameter("time", "A timestamp in Unix time format. Retrieve metric data at a single point in time. Defaults to now. Time and the combination of start, end, step are mutually exclusive.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_metric", "Sort workspaces by the specified metric. Not applicable if **start** and **end** are provided.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_type", "Sort order. One of asc, desc.").DefaultValue("desc.").DataType("string").Required(false)).
		Param(ws.QueryParameter("page", "The page number. This field paginates result data of each metric, then returns a specific page. For example, setting **page** to 2 returns the second page. It only applies to sorted metric data.").DataType("integer").Required(false)).
		Param(ws.QueryParameter("limit", "Page size, the maximum number of results in a single page. Defaults to 5.").DataType("integer").Required(false).DefaultValue("5")).
		Param(ws.QueryParameter("type", "The query type. This field can be set to 'rank' for workspaces ranking query or '' for others. Defaults to ''.").DataType("string").Required(false).DefaultValue("")).
		Metadata(restfulspec.KeyOpenAPITags, []string{constants.WorkspaceMetricsTag}).
		Writes(model.Metrics{}).
		Returns(http.StatusOK, respOK, model.Metrics{})).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("clusters/{cluster}/workspaces").
		To(h.handleWorkspaceMetricsQuery).
		Doc("Get workspace-level metric data of all workspaces of the specific cluster.").
		Param(ws.PathParameter("cluster", "The name of cluster").DataType("string").Required(true)).
		Param(ws.QueryParameter("metrics_filter", "The metric name filter consists of a regexp pattern. It specifies which metric data to return. For example, the following filter matches both workspace CPU usage and memory usage: `workspace_cpu_usage|workspace_memory_usage`. View available metrics at [kubesphere.io](https://docs.kubesphere.io/advanced-v2.0/zh-CN/api-reference/monitoring-metrics/).").DataType("string").Required(false)).
		Param(ws.QueryParameter("resources_filter", "The workspace filter consists of a regexp pattern. It specifies which workspace data to return.").DataType("string").Required(false)).
		Param(ws.QueryParameter("start", "Start time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1559347200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("end", "End time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1561939200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("step", "Time interval. Retrieve metric data at a fixed interval within the time range of start and end. It requires both **start** and **end** are provided. The format is [0-9]+[smhdwy]. Defaults to 10m (i.e. 10 min).").DataType("string").DefaultValue("10m").Required(false)).
		Param(ws.QueryParameter("time", "A timestamp in Unix time format. Retrieve metric data at a single point in time. Defaults to now. Time and the combination of start, end, step are mutually exclusive.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_metric", "Sort workspaces by the specified metric. Not applicable if **start** and **end** are provided.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_type", "Sort order. One of asc, desc.").DefaultValue("desc.").DataType("string").Required(false)).
		Param(ws.QueryParameter("page", "The page number. This field paginates result data of each metric, then returns a specific page. For example, setting **page** to 2 returns the second page. It only applies to sorted metric data.").DataType("integer").Required(false)).
		Param(ws.QueryParameter("limit", "Page size, the maximum number of results in a single page. Defaults to 5.").DataType("integer").Required(false).DefaultValue("5")).
		Metadata(restfulspec.KeyOpenAPITags, []string{constants.WorkspaceMetricsTag}).
		Writes(model.Metrics{}).
		Returns(http.StatusOK, respOK, model.Metrics{})).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/clusters/{cluster}/workspaces/{workspace}").
		To(h.handleWorkspaceMetricsQuery).
		Doc("Get workspace-level metric data for the specific workspace of the specific cluster.").
		Param(ws.PathParameter("cluster", "The name of cluster").DataType("string").Required(true)).
		Param(ws.PathParameter("workspace", "Workspace name.").DataType("string").Required(true)).
		Param(ws.QueryParameter("metrics_filter", "The metric name filter consists of a regexp pattern. It specifies which metric data to return. For example, the following filter matches both workspace CPU usage and memory usage: `workspace_cpu_usage|workspace_memory_usage`. View available metrics at [kubesphere.io](https://docs.kubesphere.io/advanced-v2.0/zh-CN/api-reference/monitoring-metrics/).").DataType("string").Required(false)).
		Param(ws.QueryParameter("start", "Start time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1559347200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("end", "End time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1561939200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("step", "Time interval. Retrieve metric data at a fixed interval within the time range of start and end. It requires both **start** and **end** are provided. The format is [0-9]+[smhdwy]. Defaults to 10m (i.e. 10 min).").DataType("string").DefaultValue("10m").Required(false)).
		Param(ws.QueryParameter("time", "A timestamp in Unix time format. Retrieve metric data at a single point in time. Defaults to now. Time and the combination of start, end, step are mutually exclusive.").DataType("string").Required(false)).
		Param(ws.QueryParameter("type", "Additional operations. Currently available types is statistics. It retrieves the total number of namespaces, devops projects, members and roles in this workspace at the moment.").DataType("string").Required(false)).
		Metadata(restfulspec.KeyOpenAPITags, []string{constants.WorkspaceMetricsTag}).
		Writes(model.Metrics{}).
		Returns(http.StatusOK, respOK, model.Metrics{})).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/namespaces").
		To(h.handleNamespaceMetricsQuery).
		Doc("Get namespace-level metric data for all cluster's namespaces.").
		Param(ws.QueryParameter("metrics_filter", "The metric name filter consists of a regexp pattern. It specifies which metric data to return. For example, the following filter matches both namespace CPU usage and memory usage: `namespace_cpu_usage|namespace_memory_usage`. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("resources_filter", "The namespace filter consists of a regexp pattern. It specifies which namespace data to return. For example, the following filter matches both namespace test and kube-system: `test|kube-system`.").DataType("string").Required(false)).
		Param(ws.QueryParameter("cluster_resources_filter", "The cluster filter consists of a regexp pattern. It specifies which cluster data to return. For example, the following filter matches both cluster host and member-a: `host|member-a`.").DataType("string").Required(false)).
		Param(ws.QueryParameter("workspace_resources_filter", "The workspace filter consists of a regexp pattern. It specifies which workspace data to return. For example, the following filter matches both workspace devops and default: `devops|default`.").DataType("string").Required(false)).
		Param(ws.QueryParameter("start", "Start time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1559347200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("end", "End time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1561939200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("step", "Time interval. Retrieve metric data at a fixed interval within the time range of start and end. It requires both **start** and **end** are provided. The format is [0-9]+[smhdwy]. Defaults to 10m (i.e. 10 min).").DataType("string").DefaultValue("10m").Required(false)).
		Param(ws.QueryParameter("time", "A timestamp in Unix time format. Retrieve metric data at a single point in time. Defaults to now. Time and the combination of start, end, step are mutually exclusive.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_metric", "Sort namespaces by the specified metric. Not applicable if **start** and **end** are provided.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_type", "Sort order. One of asc, desc.").DefaultValue("desc.").DataType("string").Required(false)).
		Param(ws.QueryParameter("page", "The page number. This field paginates result data of each metric, then returns a specific page. For example, setting **page** to 2 returns the second page. It only applies to sorted metric data.").DataType("integer").Required(false)).
		Param(ws.QueryParameter("limit", "Page size, the maximum number of results in a single page. Defaults to 5.").DataType("integer").Required(false).DefaultValue("5")).
		Param(ws.QueryParameter("type", "The query type. This field can be set to 'rank' for namespace ranking query or '' for others. Defaults to ''.").DataType("string").Required(false).DefaultValue("")).
		Metadata(restfulspec.KeyOpenAPITags, []string{constants.NamespaceMetricsTag}).
		Writes(model.Metrics{}).
		Returns(http.StatusOK, respOK, model.Metrics{})).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/clusters/{cluster}/namespaces").
		To(h.handleNamespaceMetricsQuery).
		Doc("Get namespace-level metric data for all namespaces of the specific cluster.").
		Param(ws.PathParameter("cluster", "The name of cluster").DataType("string").Required(true)).
		Param(ws.QueryParameter("metrics_filter", "The metric name filter consists of a regexp pattern. It specifies which metric data to return. For example, the following filter matches both namespace CPU usage and memory usage: `namespace_cpu_usage|namespace_memory_usage`. View available metrics at [kubesphere.io](https://docs.kubesphere.io/advanced-v2.0/zh-CN/api-reference/monitoring-metrics/).").DataType("string").Required(false)).
		Param(ws.QueryParameter("resources_filter", "The namespace filter consists of a regexp pattern. It specifies which namespace data to return. For example, the following filter matches both namespace test and kube-system: `test|kube-system`.").DataType("string").Required(false)).
		Param(ws.QueryParameter("start", "Start time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1559347200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("end", "End time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1561939200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("step", "Time interval. Retrieve metric data at a fixed interval within the time range of start and end. It requires both **start** and **end** are provided. The format is [0-9]+[smhdwy]. Defaults to 10m (i.e. 10 min).").DataType("string").DefaultValue("10m").Required(false)).
		Param(ws.QueryParameter("time", "A timestamp in Unix time format. Retrieve metric data at a single point in time. Defaults to now. Time and the combination of start, end, step are mutually exclusive.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_metric", "Sort namespaces by the specified metric. Not applicable if **start** and **end** are provided.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_type", "Sort order. One of asc, desc.").DefaultValue("desc.").DataType("string").Required(false)).
		Param(ws.QueryParameter("page", "The page number. This field paginates result data of each metric, then returns a specific page. For example, setting **page** to 2 returns the second page. It only applies to sorted metric data.").DataType("integer").Required(false)).
		Param(ws.QueryParameter("limit", "Page size, the maximum number of results in a single page. Defaults to 5.").DataType("integer").Required(false).DefaultValue("5")).
		Metadata(restfulspec.KeyOpenAPITags, []string{constants.NamespaceMetricsTag}).
		Writes(model.Metrics{}).
		Returns(http.StatusOK, respOK, model.Metrics{})).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("clusters/{cluster}/namespaces/{namespace}").
		To(h.handleNamespaceMetricsQuery).
		Doc("Get namespace-level metric data for the specific namespace of the specific cluster.").
		Param(ws.PathParameter("cluster", "The name of cluster").DataType("string").Required(true)).
		Param(ws.PathParameter("namespace", "The name of the namespace.").DataType("string").Required(true)).
		Param(ws.QueryParameter("metrics_filter", "The metric name filter consists of a regexp pattern. It specifies which metric data to return. For example, the following filter matches both namespace CPU usage and memory usage: `namespace_cpu_usage|namespace_memory_usage`. View available metrics at [kubesphere.io](https://docs.kubesphere.io/advanced-v2.0/zh-CN/api-reference/monitoring-metrics/).").DataType("string").Required(false)).
		Param(ws.QueryParameter("start", "Start time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1559347200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("end", "End time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1561939200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("step", "Time interval. Retrieve metric data at a fixed interval within the time range of start and end. It requires both **start** and **end** are provided. The format is [0-9]+[smhdwy]. Defaults to 10m (i.e. 10 min).").DataType("string").DefaultValue("10m").Required(false)).
		Param(ws.QueryParameter("time", "A timestamp in Unix time format. Retrieve metric data at a single point in time. Defaults to now. Time and the combination of start, end, step are mutually exclusive.").DataType("string").Required(false)).
		Metadata(restfulspec.KeyOpenAPITags, []string{constants.NamespaceMetricsTag}).
		Writes(model.Metrics{}).
		Returns(http.StatusOK, respOK, model.Metrics{})).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/clusters/{cluster}/workspaces/{workspace}/namespaces").
		To(h.handleNamespaceMetricsQuery).
		Doc("Get namespace-level metric data for the specific workspace's namespaces of the specific cluster.").
		Param(ws.PathParameter("cluster", "The name of cluster").DataType("string").Required(true)).
		Param(ws.PathParameter("workspace", "Workspace name.").DataType("string").Required(true)).
		Param(ws.QueryParameter("metrics_filter", "The metric name filter consists of a regexp pattern. It specifies which metric data to return. For example, the following filter matches both namespace CPU usage and memory usage: `namespace_cpu_usage|namespace_memory_usage`. View available metrics at [kubesphere.io](https://docs.kubesphere.io/advanced-v2.0/zh-CN/api-reference/monitoring-metrics/).").DataType("string").Required(false)).
		Param(ws.QueryParameter("resources_filter", "The namespace filter consists of a regexp pattern. It specifies which namespace data to return. For example, the following filter matches both namespace test and kube-system: `test|kube-system`.").DataType("string").Required(false)).
		Param(ws.QueryParameter("start", "Start time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1559347200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("end", "End time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1561939200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("step", "Time interval. Retrieve metric data at a fixed interval within the time range of start and end. It requires both **start** and **end** are provided. The format is [0-9]+[smhdwy]. Defaults to 10m (i.e. 10 min).").DataType("string").DefaultValue("10m").Required(false)).
		Param(ws.QueryParameter("time", "A timestamp in Unix time format. Retrieve metric data at a single point in time. Defaults to now. Time and the combination of start, end, step are mutually exclusive.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_metric", "Sort namespaces by the specified metric. Not applicable if **start** and **end** are provided.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_type", "Sort order. One of asc, desc.").DefaultValue("desc.").DataType("string").Required(false)).
		Param(ws.QueryParameter("page", "The page number. This field paginates result data of each metric, then returns a specific page. For example, setting **page** to 2 returns the second page. It only applies to sorted metric data.").DataType("integer").Required(false)).
		Param(ws.QueryParameter("limit", "Page size, the maximum number of results in a single page. Defaults to 5.").DataType("integer").Required(false).DefaultValue("5")).
		Metadata(restfulspec.KeyOpenAPITags, []string{constants.NamespaceMetricsTag}).
		Writes(model.Metrics{}).
		Returns(http.StatusOK, respOK, model.Metrics{})).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/workloads").
		To(h.handleWorkloadMetricsQuery).
		Doc("Get workload-level metric data for all cluster's workloads.").
		Param(ws.PathParameter("namespace", "The name of the namespace.").DataType("string").Required(true)).
		Param(ws.QueryParameter("metrics_filter", "The metric name filter consists of a regexp pattern. It specifies which metric data to return. For example, the following filter matches both workload CPU usage and memory usage: `workload_cpu_usage|workload_memory_usage`. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("resources_filter", "The workload filter consists of a regexp pattern. It specifies which workload data to return. For example, the following filter matches any workload whose name begins with prometheus: `prometheus.*`.").DataType("string").Required(false)).
		Param(ws.QueryParameter("cluster_resources_filter", "The cluster filter consists of a regexp pattern. It specifies which cluster data to return. For example, the following filter matches both cluster host and member-a: `host|member-a`.").DataType("string").Required(false)).
		Param(ws.QueryParameter("namespace_resources_filter", "The namespace filter consists of a regexp pattern. It specifies which namespace data to return. For example, the following filter matches both cluster test and default: `test|default`.").DataType("string").Required(false)).
		Param(ws.QueryParameter("start", "Start time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1559347200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("end", "End time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1561939200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("step", "Time interval. Retrieve metric data at a fixed interval within the time range of start and end. It requires both **start** and **end** are provided. The format is [0-9]+[smhdwy]. Defaults to 10m (i.e. 10 min).").DataType("string").DefaultValue("10m").Required(false)).
		Param(ws.QueryParameter("time", "A timestamp in Unix time format. Retrieve metric data at a single point in time. Defaults to now. Time and the combination of start, end, step are mutually exclusive.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_metric", "Sort workloads by the specified metric. Not applicable if **start** and **end** are provided.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_type", "Sort order. One of asc, desc.").DefaultValue("desc.").DataType("string").Required(false)).
		Param(ws.QueryParameter("page", "The page number. This field paginates result data of each metric, then returns a specific page. For example, setting **page** to 2 returns the second page. It only applies to sorted metric data.").DataType("integer").Required(false)).
		Param(ws.QueryParameter("limit", "Page size, the maximum number of results in a single page. Defaults to 5.").DataType("integer").Required(false).DefaultValue("5")).
		Param(ws.QueryParameter("type", "The query type. This field can be set to 'rank' for workloads ranking query or '' for others. Defaults to ''.").DataType("string").Required(false).DefaultValue("")).
		Metadata(restfulspec.KeyOpenAPITags, []string{constants.WorkloadMetricsTag}).
		Writes(model.Metrics{}).
		Returns(http.StatusOK, respOK, model.Metrics{})).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/workloads/{kind}").
		To(h.handleWorkloadMetricsQuery).
		Doc("Get workload-level metric data for all cluster's workloads which belongs to a specific kind.").
		Param(ws.PathParameter("kind", "Workload kind. One of deployment, daemonset, statefulset.").DataType("string").Required(true)).
		Param(ws.QueryParameter("metrics_filter", "The metric name filter consists of a regexp pattern. It specifies which metric data to return. For example, the following filter matches both workload CPU usage and memory usage: `workload_cpu_usage|workload_memory_usage`. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("resources_filter", "The workload filter consists of a regexp pattern. It specifies which workload data to return. For example, the following filter matches any workload whose name begins with prometheus: `prometheus.*`.").DataType("string").Required(false)).
		Param(ws.QueryParameter("cluster_resources_filter", "The cluster filter consists of a regexp pattern. It specifies which cluster data to return. For example, the following filter matches both cluster host and member-a: `host|member-a`.").DataType("string").Required(false)).
		Param(ws.QueryParameter("namespace_resources_filter", "The namespace filter consists of a regexp pattern. It specifies which namespace data to return. For example, the following filter matches both cluster test and default: `test|default`.").DataType("string").Required(false)).
		Param(ws.QueryParameter("start", "Start time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1559347200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("end", "End time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1561939200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("step", "Time interval. Retrieve metric data at a fixed interval within the time range of start and end. It requires both **start** and **end** are provided. The format is [0-9]+[smhdwy]. Defaults to 10m (i.e. 10 min).").DataType("string").DefaultValue("10m").Required(false)).
		Param(ws.QueryParameter("time", "A timestamp in Unix time format. Retrieve metric data at a single point in time. Defaults to now. Time and the combination of start, end, step are mutually exclusive.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_metric", "Sort workloads by the specified metric. Not applicable if **start** and **end** are provided.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_type", "Sort order. One of asc, desc.").DefaultValue("desc.").DataType("string").Required(false)).
		Param(ws.QueryParameter("page", "The page number. This field paginates result data of each metric, then returns a specific page. For example, setting **page** to 2 returns the second page. It only applies to sorted metric data.").DataType("integer").Required(false)).
		Param(ws.QueryParameter("limit", "Page size, the maximum number of results in a single page. Defaults to 5.").DataType("integer").Required(false).DefaultValue("5")).
		Param(ws.QueryParameter("type", "The query type. This field can be set to 'rank' for workloads ranking query or '' for others. Defaults to ''.").DataType("string").Required(false).DefaultValue("")).
		Metadata(restfulspec.KeyOpenAPITags, []string{constants.WorkloadMetricsTag}).
		Writes(model.Metrics{}).
		Returns(http.StatusOK, respOK, model.Metrics{})).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("clusters/{cluster}/namespaces/{namespace}/workloads").
		To(h.handleWorkloadMetricsQuery).
		Doc("Get workload-level metric data of a specific namespace's workloads.").
		Param(ws.PathParameter("cluster", "The name of cluster").DataType("string").Required(true)).
		Param(ws.PathParameter("namespace", "The name of the namespace.").DataType("string").Required(true)).
		Param(ws.QueryParameter("metrics_filter", "The metric name filter consists of a regexp pattern. It specifies which metric data to return. For example, the following filter matches both workload CPU usage and memory usage: `workload_cpu_usage|workload_memory_usage`. View available metrics at [kubesphere.io](https://docs.kubesphere.io/advanced-v2.0/zh-CN/api-reference/monitoring-metrics/).").DataType("string").Required(false)).
		Param(ws.QueryParameter("resources_filter", "The workload filter consists of a regexp pattern. It specifies which workload data to return. For example, the following filter matches any workload whose name begins with prometheus: `prometheus.*`.").DataType("string").Required(false)).
		Param(ws.QueryParameter("start", "Start time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1559347200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("end", "End time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1561939200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("step", "Time interval. Retrieve metric data at a fixed interval within the time range of start and end. It requires both **start** and **end** are provided. The format is [0-9]+[smhdwy]. Defaults to 10m (i.e. 10 min).").DataType("string").DefaultValue("10m").Required(false)).
		Param(ws.QueryParameter("time", "A timestamp in Unix time format. Retrieve metric data at a single point in time. Defaults to now. Time and the combination of start, end, step are mutually exclusive.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_metric", "Sort workloads by the specified metric. Not applicable if **start** and **end** are provided.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_type", "Sort order. One of asc, desc.").DefaultValue("desc.").DataType("string").Required(false)).
		Param(ws.QueryParameter("page", "The page number. This field paginates result data of each metric, then returns a specific page. For example, setting **page** to 2 returns the second page. It only applies to sorted metric data.").DataType("integer").Required(false)).
		Param(ws.QueryParameter("limit", "Page size, the maximum number of results in a single page. Defaults to 5.").DataType("integer").Required(false).DefaultValue("5")).
		Metadata(restfulspec.KeyOpenAPITags, []string{constants.WorkloadMetricsTag}).
		Writes(model.Metrics{}).
		Returns(http.StatusOK, respOK, model.Metrics{})).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("clusters/{cluster}/namespaces/{namespace}/workloads/{kind}").
		To(h.handleWorkloadMetricsQuery).
		Doc("Get workload-level metric data of all workloads which belongs to a specific kind.").
		Param(ws.PathParameter("cluster", "The name of cluster").DataType("string").Required(true)).
		Param(ws.PathParameter("namespace", "The name of the namespace.").DataType("string").Required(true)).
		Param(ws.PathParameter("kind", "Workload kind. One of deployment, daemonset, statefulset.").DataType("string").Required(true)).
		Param(ws.QueryParameter("metrics_filter", "The metric name filter consists of a regexp pattern. It specifies which metric data to return. For example, the following filter matches both workload CPU usage and memory usage: `workload_cpu_usage|workload_memory_usage`. View available metrics at [kubesphere.io](https://docs.kubesphere.io/advanced-v2.0/zh-CN/api-reference/monitoring-metrics/).").DataType("string").Required(false)).
		Param(ws.QueryParameter("resources_filter", "The workload filter consists of a regexp pattern. It specifies which workload data to return. For example, the following filter matches any workload whose name begins with prometheus: `prometheus.*`.").DataType("string").Required(false)).
		Param(ws.QueryParameter("start", "Start time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1559347200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("end", "End time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1561939200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("step", "Time interval. Retrieve metric data at a fixed interval within the time range of start and end. It requires both **start** and **end** are provided. The format is [0-9]+[smhdwy]. Defaults to 10m (i.e. 10 min).").DataType("string").DefaultValue("10m").Required(false)).
		Param(ws.QueryParameter("time", "A timestamp in Unix time format. Retrieve metric data at a single point in time. Defaults to now. Time and the combination of start, end, step are mutually exclusive.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_metric", "Sort workloads by the specified metric. Not applicable if **start** and **end** are provided.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_type", "Sort order. One of asc, desc.").DefaultValue("desc.").DataType("string").Required(false)).
		Param(ws.QueryParameter("page", "The page number. This field paginates result data of each metric, then returns a specific page. For example, setting **page** to 2 returns the second page. It only applies to sorted metric data.").DataType("integer").Required(false)).
		Param(ws.QueryParameter("limit", "Page size, the maximum number of results in a single page. Defaults to 5.").DataType("integer").Required(false).DefaultValue("5")).
		Metadata(restfulspec.KeyOpenAPITags, []string{constants.WorkloadMetricsTag}).
		Writes(model.Metrics{}).
		Returns(http.StatusOK, respOK, model.Metrics{})).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/pods").
		To(h.handlePodMetricsQuery).
		Doc("Get pod-level metric data for all cluster's pods.").
		Param(ws.QueryParameter("metrics_filter", "The metric name filter consists of a regexp pattern. It specifies which metric data to return. For example, the following filter matches both pod CPU usage and memory usage: `pod_cpu_usage|pod_memory_usage`. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("resources_filter", "The pod filter consists of a regexp pattern. It specifies which pod data to return. For example, the following filter matches any pod whose name begins with redis: `redis.*`.").DataType("string").Required(false)).
		Param(ws.QueryParameter("cluster_resources_filter", "The cluster filter consists of a regexp pattern. It specifies which cluster data to return. For example, the following filter matches both cluster host and member-a: `host|member-a`.").DataType("string").Required(false)).
		Param(ws.QueryParameter("workspace_resources_filter", "The workspace filter consists of a regexp pattern. It specifies which workspace data to return. For example, the following filter matches both workspace devops and default: `devops|default`.").DataType("string").Required(false)).
		Param(ws.QueryParameter("namespace_resources_filter", "The namespace filter consists of a regexp pattern. It specifies which namespace data to return. For example, the following filter matches both cluster test and default: `test|default`.").DataType("string").Required(false)).
		Param(ws.QueryParameter("start", "Start time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1559347200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("end", "End time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1561939200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("step", "Time interval. Retrieve metric data at a fixed interval within the time range of start and end. It requires both **start** and **end** are provided. The format is [0-9]+[smhdwy]. Defaults to 10m (i.e. 10 min).").DataType("string").DefaultValue("10m").Required(false)).
		Param(ws.QueryParameter("time", "A timestamp in Unix time format. Retrieve metric data at a single point in time. Defaults to now. Time and the combination of start, end, step are mutually exclusive.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_metric", "Sort pods by the specified metric. Not applicable if **start** and **end** are provided.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_type", "Sort order. One of asc, desc.").DefaultValue("desc.").DataType("string").Required(false)).
		Param(ws.QueryParameter("page", "The page number. This field paginates result data of each metric, then returns a specific page. For example, setting **page** to 2 returns the second page. It only applies to sorted metric data.").DataType("integer").Required(false)).
		Param(ws.QueryParameter("limit", "Page size, the maximum number of results in a single page. Defaults to 5.").DataType("integer").Required(false).DefaultValue("5")).
		Param(ws.QueryParameter("type", "The query type. This field can be set to 'rank' for pods ranking query or '' for others. Defaults to ''.").DataType("string").Required(false).DefaultValue("")).
		Metadata(restfulspec.KeyOpenAPITags, []string{constants.PodMetricsTag}).
		Writes(model.Metrics{}).
		Returns(http.StatusOK, respOK, model.Metrics{})).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/clusters/{cluster}/pods").
		To(h.handlePodMetricsQuery).
		Doc("Get pod-level metric data of the specific cluster's pods.").
		Param(ws.PathParameter("cluster", "The name of cluster").DataType("string").Required(true)).
		Param(ws.QueryParameter("metrics_filter", "The metric name filter consists of a regexp pattern. It specifies which metric data to return. For example, the following filter matches both pod CPU usage and memory usage: `pod_cpu_usage|pod_memory_usage`. View available metrics at [kubesphere.io](https://docs.kubesphere.io/advanced-v2.0/zh-CN/api-reference/monitoring-metrics/).").DataType("string").Required(false)).
		Param(ws.QueryParameter("namespaced_resources_filter", "Specifies a namespaced resources filter in `<namespace>/<pod_name>|<namespace>/<pod_name>` format. For example, a namespaced resources filter like `ns1/pod1|ns2/pod2` will request the data of pod1 in ns1 together with pod2 in ns2.").DataType("string").Required(false)).
		Param(ws.QueryParameter("start", "Start time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1559347200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("end", "End time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1561939200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("step", "Time interval. Retrieve metric data at a fixed interval within the time range of start and end. It requires both **start** and **end** are provided. The format is [0-9]+[smhdwy]. Defaults to 10m (i.e. 10 min).").DataType("string").DefaultValue("10m").Required(false)).
		Param(ws.QueryParameter("time", "A timestamp in Unix time format. Retrieve metric data at a single point in time. Defaults to now. Time and the combination of start, end, step are mutually exclusive.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_metric", "Sort pods by the specified metric. Not applicable if **start** and **end** are provided.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_type", "Sort order. One of asc, desc.").DefaultValue("desc.").DataType("string").Required(false)).
		Param(ws.QueryParameter("page", "The page number. This field paginates result data of each metric, then returns a specific page. For example, setting **page** to 2 returns the second page. It only applies to sorted metric data.").DataType("integer").Required(false)).
		Param(ws.QueryParameter("limit", "Page size, the maximum number of results in a single page. Defaults to 5.").DataType("integer").Required(false).DefaultValue("5")).
		Metadata(restfulspec.KeyOpenAPITags, []string{constants.PodMetricsTag}).
		Writes(model.Metrics{}).
		Returns(http.StatusOK, respOK, model.Metrics{})).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/clusters/{cluster}/namespaces/{namespace}/pods").
		To(h.handlePodMetricsQuery).
		Doc("Get pod-level metric data for the specific namespace's pods of the specific cluster.").
		Param(ws.PathParameter("cluster", "The name of cluster").DataType("string").Required(true)).
		Param(ws.PathParameter("namespace", "The name of the namespace.").DataType("string").Required(true)).
		Param(ws.QueryParameter("metrics_filter", "The metric name filter consists of a regexp pattern. It specifies which metric data to return. For example, the following filter matches both pod CPU usage and memory usage: `pod_cpu_usage|pod_memory_usage`. View available metrics at [kubesphere.io](https://docs.kubesphere.io/advanced-v2.0/zh-CN/api-reference/monitoring-metrics/).").DataType("string").Required(false)).
		Param(ws.QueryParameter("resources_filter", "The pod filter consists of a regexp pattern. It specifies which pod data to return. For example, the following filter matches any pod whose name begins with redis: `redis.*`.").DataType("string").Required(false)).
		Param(ws.QueryParameter("start", "Start time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1559347200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("end", "End time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1561939200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("step", "Time interval. Retrieve metric data at a fixed interval within the time range of start and end. It requires both **start** and **end** are provided. The format is [0-9]+[smhdwy]. Defaults to 10m (i.e. 10 min).").DataType("string").DefaultValue("10m").Required(false)).
		Param(ws.QueryParameter("time", "A timestamp in Unix time format. Retrieve metric data at a single point in time. Defaults to now. Time and the combination of start, end, step are mutually exclusive.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_metric", "Sort pods by the specified metric. Not applicable if **start** and **end** are provided.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_type", "Sort order. One of asc, desc.").DefaultValue("desc.").DataType("string").Required(false)).
		Param(ws.QueryParameter("page", "The page number. This field paginates result data of each metric, then returns a specific page. For example, setting **page** to 2 returns the second page. It only applies to sorted metric data.").DataType("integer").Required(false)).
		Param(ws.QueryParameter("limit", "Page size, the maximum number of results in a single page. Defaults to 5.").DataType("integer").Required(false).DefaultValue("5")).
		Metadata(restfulspec.KeyOpenAPITags, []string{constants.PodMetricsTag}).
		Writes(model.Metrics{}).
		Returns(http.StatusOK, respOK, model.Metrics{})).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/clusters/{cluster}/namespaces/{namespace}/pods/{pod}").
		To(h.handlePodMetricsQuery).
		Doc("Get pod-level metric data of a specific pod. Navigate to the pod by the pod's cluster and namespace.").
		Param(ws.PathParameter("cluster", "The name of cluster").DataType("string").Required(true)).
		Param(ws.PathParameter("namespace", "The name of the namespace.").DataType("string").Required(true)).
		Param(ws.PathParameter("pod", "Pod name.").DataType("string").Required(true)).
		Param(ws.QueryParameter("metrics_filter", "The metric name filter consists of a regexp pattern. It specifies which metric data to return. For example, the following filter matches both pod CPU usage and memory usage: `pod_cpu_usage|pod_memory_usage`. View available metrics at [kubesphere.io](https://docs.kubesphere.io/advanced-v2.0/zh-CN/api-reference/monitoring-metrics/).").DataType("string").Required(false)).
		Param(ws.QueryParameter("start", "Start time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1559347200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("end", "End time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1561939200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("step", "Time interval. Retrieve metric data at a fixed interval within the time range of start and end. It requires both **start** and **end** are provided. The format is [0-9]+[smhdwy]. Defaults to 10m (i.e. 10 min).").DataType("string").DefaultValue("10m").Required(false)).
		Param(ws.QueryParameter("time", "A timestamp in Unix time format. Retrieve metric data at a single point in time. Defaults to now. Time and the combination of start, end, step are mutually exclusive.").DataType("string").Required(false)).
		Metadata(restfulspec.KeyOpenAPITags, []string{constants.PodMetricsTag}).
		Writes(model.Metrics{}).
		Returns(http.StatusOK, respOK, model.Metrics{})).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/clusters/{cluster}/namespaces/{namespace}/workloads/{kind}/{workload}/pods").
		To(h.handlePodMetricsQuery).
		Doc("Get pod-level metric data of a specific workload's pods. Navigate to the workload by the cluster and namespace.").
		Param(ws.PathParameter("cluster", "The name of cluster").DataType("string").Required(true)).
		Param(ws.PathParameter("namespace", "The name of the namespace.").DataType("string").Required(true)).
		Param(ws.PathParameter("kind", "Workload kind. One of deployment, daemonset, statefulset.").DataType("string").Required(true)).
		Param(ws.PathParameter("workload", "Workload name.").DataType("string").Required(true)).
		Param(ws.QueryParameter("metrics_filter", "The metric name filter consists of a regexp pattern. It specifies which metric data to return. For example, the following filter matches both pod CPU usage and memory usage: `pod_cpu_usage|pod_memory_usage`. View available metrics at [kubesphere.io](https://docs.kubesphere.io/advanced-v2.0/zh-CN/api-reference/monitoring-metrics/).").DataType("string").Required(false)).
		Param(ws.QueryParameter("resources_filter", "The pod filter consists of a regexp pattern. It specifies which pod data to return. For example, the following filter matches any pod whose name begins with redis: `redis.*`.").DataType("string").Required(false)).
		Param(ws.QueryParameter("start", "Start time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1559347200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("end", "End time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1561939200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("step", "Time interval. Retrieve metric data at a fixed interval within the time range of start and end. It requires both **start** and **end** are provided. The format is [0-9]+[smhdwy]. Defaults to 10m (i.e. 10 min).").DataType("string").DefaultValue("10m").Required(false)).
		Param(ws.QueryParameter("time", "A timestamp in Unix time format. Retrieve metric data at a single point in time. Defaults to now. Time and the combination of start, end, step are mutually exclusive.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_metric", "Sort pods by the specified metric. Not applicable if **start** and **end** are provided.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_type", "Sort order. One of asc, desc.").DefaultValue("desc.").DataType("string").Required(false)).
		Param(ws.QueryParameter("page", "The page number. This field paginates result data of each metric, then returns a specific page. For example, setting **page** to 2 returns the second page. It only applies to sorted metric data.").DataType("integer").Required(false)).
		Param(ws.QueryParameter("limit", "Page size, the maximum number of results in a single page. Defaults to 5.").DataType("integer").Required(false).DefaultValue("5")).
		Metadata(restfulspec.KeyOpenAPITags, []string{constants.PodMetricsTag}).
		Writes(model.Metrics{}).
		Returns(http.StatusOK, respOK, model.Metrics{})).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/clusters/{cluster}/nodes/{node}/pods").
		To(h.handlePodMetricsQuery).
		Doc("Get pod-level metric data of all pods on a specific node of the specific cluster.").
		Param(ws.PathParameter("cluster", "The name of cluster").DataType("string").Required(true)).
		Param(ws.PathParameter("node", "Node name.").DataType("string").Required(true)).
		Param(ws.QueryParameter("metrics_filter", "The metric name filter consists of a regexp pattern. It specifies which metric data to return. For example, the following filter matches both pod CPU usage and memory usage: `pod_cpu_usage|pod_memory_usage`. View available metrics at [kubesphere.io](https://docs.kubesphere.io/advanced-v2.0/zh-CN/api-reference/monitoring-metrics/).").DataType("string").Required(false)).
		Param(ws.QueryParameter("resources_filter", "The pod filter consists of a regexp pattern. It specifies which pod data to return. For example, the following filter matches any pod whose name begins with redis: `redis.*`.").DataType("string").Required(false)).
		Param(ws.QueryParameter("namespaced_resources_filter", "Specifies a namespaced resources filter in `<namespace>/<pod_name>|<namespace>/<pod_name>` format. For example, a namespaced resources filter like `ns1/pod1|ns2/pod2` will request the data of pod1 in ns1 together with pod2 in ns2.").DataType("string").Required(false)).
		Param(ws.QueryParameter("start", "Start time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1559347200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("end", "End time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1561939200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("step", "Time interval. Retrieve metric data at a fixed interval within the time range of start and end. It requires both **start** and **end** are provided. The format is [0-9]+[smhdwy]. Defaults to 10m (i.e. 10 min).").DataType("string").DefaultValue("10m").Required(false)).
		Param(ws.QueryParameter("time", "A timestamp in Unix time format. Retrieve metric data at a single point in time. Defaults to now. Time and the combination of start, end, step are mutually exclusive.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_metric", "Sort pods by the specified metric. Not applicable if **start** and **end** are provided.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_type", "Sort order. One of asc, desc.").DefaultValue("desc.").DataType("string").Required(false)).
		Param(ws.QueryParameter("page", "The page number. This field paginates result data of each metric, then returns a specific page. For example, setting **page** to 2 returns the second page. It only applies to sorted metric data.").DataType("integer").Required(false)).
		Param(ws.QueryParameter("limit", "Page size, the maximum number of results in a single page. Defaults to 5.").DataType("integer").Required(false).DefaultValue("5")).
		Metadata(restfulspec.KeyOpenAPITags, []string{constants.PodMetricsTag}).
		Writes(model.Metrics{}).
		Returns(http.StatusOK, respOK, model.Metrics{})).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/containers").
		To(h.handleContainerMetricsQuery).
		Doc("Get container-level metric data for all cluster's containers").
		Param(ws.QueryParameter("metrics_filter", "The metric name filter consists of a regexp pattern. It specifies which metric data to return. For example, the following filter matches both container CPU usage and memory usage: `container_cpu_usage|container_memory_usage`. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("resources_filter", "The container filter consists of a regexp pattern. It specifies which container data to return. For example, the following filter matches container prometheus and prometheus-config-reloader: `prometheus|prometheus-config-reloader`.").DataType("string").Required(false)).
		Param(ws.QueryParameter("cluster_resources_filter", "The cluster filter consists of a regexp pattern. It specifies which cluster data to return. For example, the following filter matches both cluster host and member-a: `host|member-a`.").DataType("string").Required(false)).
		Param(ws.QueryParameter("namespace_resources_filter", "The namespace filter consists of a regexp pattern. It specifies which namespace data to return. For example, the following filter matches both namespace test and default: `test|default`.").DataType("string").Required(false)).
		Param(ws.QueryParameter("pod_resources_filter", "The pod filter consists of a regexp pattern. It specifies which pod data to return. For example, the following filter matches both pod sts-kafka-0 and nginx-5dc869b4fb-26v4b: `sts-kafka-0|nginx-5dc869b4fb-26v4b`.").DataType("string").Required(false)).
		Param(ws.QueryParameter("start", "Start time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1559347200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("end", "End time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1561939200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("step", "Time interval. Retrieve metric data at a fixed interval within the time range of start and end. It requires both **start** and **end** are provided. The format is [0-9]+[smhdwy]. Defaults to 10m (i.e. 10 min).").DataType("string").DefaultValue("10m").Required(false)).
		Param(ws.QueryParameter("time", "A timestamp in Unix time format. Retrieve metric data at a single point in time. Defaults to now. Time and the combination of start, end, step are mutually exclusive.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_metric", "Sort containers by the specified metric. Not applicable if **start** and **end** are provided.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_type", "Sort order. One of asc, desc.").DefaultValue("desc.").DataType("string").Required(false)).
		Param(ws.QueryParameter("page", "The page number. This field paginates result data of each metric, then returns a specific page. For example, setting **page** to 2 returns the second page. It only applies to sorted metric data.").DataType("integer").Required(false)).
		Param(ws.QueryParameter("limit", "Page size, the maximum number of results in a single page. Defaults to 5.").DataType("integer").Required(false).DefaultValue("5")).
		Param(ws.QueryParameter("type", "The query type. This field can be set to 'rank' for cluster ranking query or '' for others. Defaults to ''.").DataType("string").Required(false).DefaultValue("")).
		Metadata(restfulspec.KeyOpenAPITags, []string{constants.ContainerMetricsTag}).
		Writes(model.Metrics{}).
		Returns(http.StatusOK, respOK, model.Metrics{})).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/clusters/{cluster}/namespaces/{namespace}/pods/{pod}/containers").
		To(h.handleContainerMetricsQuery).
		Doc("Get container-level metric data of a specific pod's containers. Navigate to the pod by the pod's cluster and namespace.").
		Param(ws.PathParameter("cluster", "The name of cluster").DataType("string").Required(true)).
		Param(ws.PathParameter("namespace", "The name of the namespace.").DataType("string").Required(true)).
		Param(ws.PathParameter("pod", "Pod name.").DataType("string").Required(true)).
		Param(ws.QueryParameter("metrics_filter", "The metric name filter consists of a regexp pattern. It specifies which metric data to return. For example, the following filter matches both container CPU usage and memory usage: `container_cpu_usage|container_memory_usage`. View available metrics at [kubesphere.io](https://docs.kubesphere.io/advanced-v2.0/zh-CN/api-reference/monitoring-metrics/).").DataType("string").Required(false)).
		Param(ws.QueryParameter("resources_filter", "The container filter consists of a regexp pattern. It specifies which container data to return. For example, the following filter matches container prometheus and prometheus-config-reloader: `prometheus|prometheus-config-reloader`.").DataType("string").Required(false)).
		Param(ws.QueryParameter("start", "Start time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1559347200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("end", "End time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1561939200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("step", "Time interval. Retrieve metric data at a fixed interval within the time range of start and end. It requires both **start** and **end** are provided. The format is [0-9]+[smhdwy]. Defaults to 10m (i.e. 10 min).").DataType("string").DefaultValue("10m").Required(false)).
		Param(ws.QueryParameter("time", "A timestamp in Unix time format. Retrieve metric data at a single point in time. Defaults to now. Time and the combination of start, end, step are mutually exclusive.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_metric", "Sort containers by the specified metric. Not applicable if **start** and **end** are provided.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_type", "Sort order. One of asc, desc.").DefaultValue("desc.").DataType("string").Required(false)).
		Param(ws.QueryParameter("page", "The page number. This field paginates result data of each metric, then returns a specific page. For example, setting **page** to 2 returns the second page. It only applies to sorted metric data.").DataType("integer").Required(false)).
		Param(ws.QueryParameter("limit", "Page size, the maximum number of results in a single page. Defaults to 5.").DataType("integer").Required(false).DefaultValue("5")).
		Metadata(restfulspec.KeyOpenAPITags, []string{constants.ContainerMetricsTag}).
		Writes(model.Metrics{}).
		Returns(http.StatusOK, respOK, model.Metrics{})).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/clusters/{cluster}/namespaces/{namespace}/pods/{pod}/containers/{container}").
		To(h.handleContainerMetricsQuery).
		Doc("Get container-level metric data of a specific container. Navigate to the container by the pod's cluster, namespace and name ").
		Param(ws.PathParameter("cluster", "The name of cluster").DataType("string").Required(true)).
		Param(ws.PathParameter("namespace", "The name of the namespace.").DataType("string").Required(true)).
		Param(ws.PathParameter("pod", "Pod name.").DataType("string").Required(true)).
		Param(ws.PathParameter("container", "Container name.").DataType("string").Required(true)).
		Param(ws.QueryParameter("metrics_filter", "The metric name filter consists of a regexp pattern. It specifies which metric data to return. For example, the following filter matches both container CPU usage and memory usage: `container_cpu_usage|container_memory_usage`. View available metrics at [kubesphere.io](https://docs.kubesphere.io/advanced-v2.0/zh-CN/api-reference/monitoring-metrics/).").DataType("string").Required(false)).
		Param(ws.QueryParameter("start", "Start time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1559347200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("end", "End time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1561939200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("step", "Time interval. Retrieve metric data at a fixed interval within the time range of start and end. It requires both **start** and **end** are provided. The format is [0-9]+[smhdwy]. Defaults to 10m (i.e. 10 min).").DataType("string").DefaultValue("10m").Required(false)).
		Param(ws.QueryParameter("time", "A timestamp in Unix time format. Retrieve metric data at a single point in time. Defaults to now. Time and the combination of start, end, step are mutually exclusive.").DataType("string").Required(false)).
		Metadata(restfulspec.KeyOpenAPITags, []string{constants.ContainerMetricsTag}).
		Writes(model.Metrics{}).
		Returns(http.StatusOK, respOK, model.Metrics{})).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/persistentvolumeclaims").
		To(h.handlePVCMetricsQuery).
		Doc("Get PVC-level metric data for all cluster's persistentvolumeclaims").
		Param(ws.QueryParameter("metrics_filter", "The metric name filter consists of a regexp pattern. It specifies which metric data to return. For example, the following filter matches both PVC available and used inodes: `pvc_inodes_available|pvc_inodes_used`. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("resources_filter", "The PVC filter consists of a regexp pattern. It specifies which PVC data to return. For example, the following filter matches any pod whose name begins with redis: `redis.*`.").DataType("string").Required(false)).
		Param(ws.QueryParameter("cluster_resources_filter", "The cluster filter consists of a regexp pattern. It specifies which cluster data to return. For example, the following filter matches both cluster host and member-a: `host|member-a`.").DataType("string").Required(false)).
		Param(ws.QueryParameter("namespace_resources_filter", "The namespace filter consists of a regexp pattern. It specifies which namespace data to return. For example, the following filter matches both namespace test and default: `test|default`.").DataType("string").Required(false)).
		Param(ws.QueryParameter("storageclass_resources_filter", "The storageclass filter consists of a regexp pattern. It specifies which storageclass data to return. For example, the following filter matches both namespace local and default: `local|default`.").DataType("string").Required(false)).
		Param(ws.QueryParameter("start", "Start time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1559347200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("end", "End time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1561939200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("step", "Time interval. Retrieve metric data at a fixed interval within the time range of start and end. It requires both **start** and **end** are provided. The format is [0-9]+[smhdwy]. Defaults to 10m (i.e. 10 min).").DataType("string").DefaultValue("10m").Required(false)).
		Param(ws.QueryParameter("time", "A timestamp in Unix time format. Retrieve metric data at a single point in time. Defaults to now. Time and the combination of start, end, step are mutually exclusive.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_metric", "Sort PVCs by the specified metric. Not applicable if **start** and **end** are provided.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_type", "Sort order. One of asc, desc.").DefaultValue("desc.").DataType("string").Required(false)).
		Param(ws.QueryParameter("page", "The page number. This field paginates result data of each metric, then returns a specific page. For example, setting **page** to 2 returns the second page. It only applies to sorted metric data.").DataType("integer").Required(false)).
		Param(ws.QueryParameter("limit", "Page size, the maximum number of results in a single page. Defaults to 5.").DataType("integer").Required(false).DefaultValue("5")).
		Param(ws.QueryParameter("type", "The query type. This field can be set to 'rank' for cluster ranking query or '' for others. Defaults to ''.").DataType("string").Required(false).DefaultValue("")).
		Metadata(restfulspec.KeyOpenAPITags, []string{constants.PVCMetricsTag}).
		Writes(model.Metrics{}).
		Returns(http.StatusOK, respOK, model.Metrics{})).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/clusters/{cluster}/storageclasses/{storageclass}/persistentvolumeclaims").
		To(h.handlePVCMetricsQuery).
		Doc("Get PVC-level metric data for the specific storageclass's PVCs of the specified cluster.").
		Param(ws.PathParameter("cluster", "The name of cluster").DataType("string").Required(true)).
		Param(ws.PathParameter("storageclass", "The name of the storageclass.").DataType("string").Required(true)).
		Param(ws.QueryParameter("metrics_filter", "The metric name filter consists of a regexp pattern. It specifies which metric data to return. For example, the following filter matches both PVC available and used inodes: `pvc_inodes_available|pvc_inodes_used`. View available metrics at [kubesphere.io](https://docs.kubesphere.io/advanced-v2.0/zh-CN/api-reference/monitoring-metrics/).").DataType("string").Required(false)).
		Param(ws.QueryParameter("resources_filter", "The PVC filter consists of a regexp pattern. It specifies which PVC data to return. For example, the following filter matches any pod whose name begins with redis: `redis.*`.").DataType("string").Required(false)).
		Param(ws.QueryParameter("start", "Start time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1559347200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("end", "End time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1561939200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("step", "Time interval. Retrieve metric data at a fixed interval within the time range of start and end. It requires both **start** and **end** are provided. The format is [0-9]+[smhdwy]. Defaults to 10m (i.e. 10 min).").DataType("string").DefaultValue("10m").Required(false)).
		Param(ws.QueryParameter("time", "A timestamp in Unix time format. Retrieve metric data at a single point in time. Defaults to now. Time and the combination of start, end, step are mutually exclusive.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_metric", "Sort PVCs by the specified metric. Not applicable if **start** and **end** are provided.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_type", "Sort order. One of asc, desc.").DefaultValue("desc.").DataType("string").Required(false)).
		Param(ws.QueryParameter("page", "The page number. This field paginates result data of each metric, then returns a specific page. For example, setting **page** to 2 returns the second page. It only applies to sorted metric data.").DataType("integer").Required(false)).
		Param(ws.QueryParameter("limit", "Page size, the maximum number of results in a single page. Defaults to 5.").DataType("integer").Required(false).DefaultValue("5")).
		Metadata(restfulspec.KeyOpenAPITags, []string{constants.PVCMetricsTag}).
		Writes(model.Metrics{}).
		Returns(http.StatusOK, respOK, model.Metrics{})).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/clusters/{cluster}/namespaces/{namespace}/persistentvolumeclaims").
		To(h.handlePVCMetricsQuery).
		Doc("Get PVC-level metric data for the specific namespace's PVCs of the specified cluster.").
		Param(ws.PathParameter("cluster", "The name of cluster").DataType("string").Required(true)).
		Param(ws.PathParameter("namespace", "The name of the namespace.").DataType("string").Required(true)).
		Param(ws.QueryParameter("metrics_filter", "The metric name filter consists of a regexp pattern. It specifies which metric data to return. For example, the following filter matches both PVC available and used inodes: `pvc_inodes_available|pvc_inodes_used`. View available metrics at [kubesphere.io](https://docs.kubesphere.io/advanced-v2.0/zh-CN/api-reference/monitoring-metrics/).").DataType("string").Required(false)).
		Param(ws.QueryParameter("resources_filter", "The PVC filter consists of a regexp pattern. It specifies which PVC data to return. For example, the following filter matches any pod whose name begins with redis: `redis.*`.").DataType("string").Required(false)).
		Param(ws.QueryParameter("start", "Start time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1559347200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("end", "End time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1561939200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("step", "Time interval. Retrieve metric data at a fixed interval within the time range of start and end. It requires both **start** and **end** are provided. The format is [0-9]+[smhdwy]. Defaults to 10m (i.e. 10 min).").DataType("string").DefaultValue("10m").Required(false)).
		Param(ws.QueryParameter("time", "A timestamp in Unix time format. Retrieve metric data at a single point in time. Defaults to now. Time and the combination of start, end, step are mutually exclusive.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_metric", "Sort PVCs by the specified metric. Not applicable if **start** and **end** are provided.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_type", "Sort order. One of asc, desc.").DefaultValue("desc.").DataType("string").Required(false)).
		Param(ws.QueryParameter("page", "The page number. This field paginates result data of each metric, then returns a specific page. For example, setting **page** to 2 returns the second page. It only applies to sorted metric data.").DataType("integer").Required(false)).
		Param(ws.QueryParameter("limit", "Page size, the maximum number of results in a single page. Defaults to 5.").DataType("integer").Required(false).DefaultValue("5")).
		Metadata(restfulspec.KeyOpenAPITags, []string{constants.PVCMetricsTag}).
		Writes(model.Metrics{}).
		Returns(http.StatusOK, respOK, model.Metrics{})).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/clusters/{cluster}/namespaces/{namespace}/persistentvolumeclaims/{pvc}").
		To(h.handlePVCMetricsQuery).
		Doc("Get PVC-level metric data of the specific PVC. Navigate to the PVC by the PVC's cluster and namespace.").
		Param(ws.PathParameter("cluster", "The name of cluster").DataType("string").Required(true)).
		Param(ws.PathParameter("namespace", "The name of the namespace.").DataType("string").Required(true)).
		Param(ws.PathParameter("pvc", "PVC name.").DataType("string").Required(true)).
		Param(ws.QueryParameter("metrics_filter", "The metric name filter consists of a regexp pattern. It specifies which metric data to return. For example, the following filter matches both PVC available and used inodes: `pvc_inodes_available|pvc_inodes_used`. View available metrics at [kubesphere.io](https://docs.kubesphere.io/advanced-v2.0/zh-CN/api-reference/monitoring-metrics/).").DataType("string").Required(false)).
		Param(ws.QueryParameter("start", "Start time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1559347200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("end", "End time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1561939200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("step", "Time interval. Retrieve metric data at a fixed interval within the time range of start and end. It requires both **start** and **end** are provided. The format is [0-9]+[smhdwy]. Defaults to 10m (i.e. 10 min).").DataType("string").DefaultValue("10m").Required(false)).
		Param(ws.QueryParameter("time", "A timestamp in Unix time format. Retrieve metric data at a single point in time. Defaults to now. Time and the combination of start, end, step are mutually exclusive.").DataType("string").Required(false)).
		Metadata(restfulspec.KeyOpenAPITags, []string{constants.PVCMetricsTag}).
		Writes(model.Metrics{}).
		Returns(http.StatusOK, respOK, model.Metrics{})).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/ingresses").
		To(h.handleIngressMetricsQuery).
		Doc("Get Ingress-level metric data for all cluster's ingresses.").
		Param(ws.QueryParameter("metrics_filter", "The metric name filter consists of a regexp pattern. It specifies which metric data to return. For example, the following filter matches both PVC available and used inodes: `pvc_inodes_available|pvc_inodes_used`. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("resources_filter", "The PVC filter consists of a regexp pattern. It specifies which PVC data to return. For example, the following filter matches any pod whose name begins with redis: `redis.*`.").DataType("string").Required(false)).
		Param(ws.QueryParameter("cluster_resources_filter", "The cluster filter consists of a regexp pattern. It specifies which cluster data to return. For example, the following filter matches both cluster host and member-a: `host|member-a`.").DataType("string").Required(false)).
		Param(ws.QueryParameter("namespace_resources_filter", "The namespace filter consists of a regexp pattern. It specifies which namespace data to return. For example, the following filter matches both namespace test and default: `test|default`.").DataType("string").Required(false)).
		Param(ws.QueryParameter("job", "The job name filter. Ingress could be served by multi Ingress controllers, The job filters metric from a specific controller.").DataType("string").Required(true)).
		Param(ws.QueryParameter("start", "Start time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1559347200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("end", "End time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1561939200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("step", "Time interval. Retrieve metric data at a fixed interval within the time range of start and end. It requires both **start** and **end** are provided. The format is [0-9]+[smhdwy]. Defaults to 10m (i.e. 10 min).").DataType("string").DefaultValue("10m").Required(false)).
		Param(ws.QueryParameter("time", "A timestamp in Unix time format. Retrieve metric data at a single point in time. Defaults to now. Time and the combination of start, end, step are mutually exclusive.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_metric", "Sort ingresses by the specified metric. Not applicable if **start** and **end** are provided.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_type", "Sort order. One of asc, desc.").DefaultValue("desc.").DataType("string").Required(false)).
		Param(ws.QueryParameter("page", "The page number. This field paginates result data of each metric, then returns a specific page. For example, setting **page** to 2 returns the second page. It only applies to sorted metric data.").DataType("integer").Required(false)).
		Param(ws.QueryParameter("limit", "Page size, the maximum number of results in a single page. Defaults to 5.").DataType("integer").Required(false).DefaultValue("5")).
		Param(ws.QueryParameter("type", "The query type. This field can be set to 'rank' for cluster ranking query or '' for others. Defaults to ''.").DataType("string").Required(false).DefaultValue("")).
		Metadata(restfulspec.KeyOpenAPITags, []string{constants.IngressMetricsTag}).
		Writes(model.Metrics{}).
		Returns(http.StatusOK, respOK, model.Metrics{})).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/clusters/{cluster}/namespaces/{namespace}/ingresses").
		To(h.handleIngressMetricsQuery).
		Doc("Get Ingress-level metric data for the specific namespace's Ingresses of the specified cluster.").
		Param(ws.PathParameter("cluster", "The name of cluster").DataType("string").Required(true)).
		Param(ws.PathParameter("namespace", "The name of the namespace.").DataType("string").Required(true)).
		Param(ws.QueryParameter("job", "The job name filter. Ingress could be served by multi Ingress controllers, The job filters metric from a specific controller.").DataType("string").Required(true)).
		Param(ws.QueryParameter("pod", "The pod name filter.").DataType("string")).
		Param(ws.QueryParameter("duration", "The duration is the time window of Range Vector. The format is [0-9]+[smhdwy]. Defaults to 5m (i.e. 5 min).").DataType("string").Required(false)).
		Param(ws.QueryParameter("metrics_filter", "The metric name filter consists of a regexp pattern. It specifies which metric data to return. For example, the following filter matches both PVC available and used inodes: `pvc_inodes_available|pvc_inodes_used`. View available metrics at [kubesphere.io](https://docs.kubesphere.io/advanced-v2.0/zh-CN/api-reference/monitoring-metrics/).").DataType("string").Required(false)).
		Param(ws.QueryParameter("resources_filter", "The PVC filter consists of a regexp pattern. It specifies which PVC data to return. For example, the following filter matches any pod whose name begins with redis: `redis.*`.").DataType("string").Required(false)).
		Param(ws.QueryParameter("start", "Start time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1559347200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("end", "End time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1561939200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("step", "Time interval. Retrieve metric data at a fixed interval within the time range of start and end. It requires both **start** and **end** are provided. The format is [0-9]+[smhdwy]. Defaults to 10m (i.e. 10 min).").DataType("string").DefaultValue("10m").Required(false)).
		Param(ws.QueryParameter("time", "A timestamp in Unix time format. Retrieve metric data at a single point in time. Defaults to now. Time and the combination of start, end, step are mutually exclusive.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_metric", "Sort PVCs by the specified metric. Not applicable if **start** and **end** are provided.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_type", "Sort order. One of asc, desc.").DefaultValue("desc.").DataType("string").Required(false)).
		Param(ws.QueryParameter("page", "The page number. This field paginates result data of each metric, then returns a specific page. For example, setting **page** to 2 returns the second page. It only applies to sorted metric data.").DataType("integer").Required(false)).
		Param(ws.QueryParameter("limit", "Page size, the maximum number of results in a single page. Defaults to 5.").DataType("integer").Required(false).DefaultValue("5")).
		Metadata(restfulspec.KeyOpenAPITags, []string{constants.IngressMetricsTag}).
		Writes(model.Metrics{}).
		Returns(http.StatusOK, respOK, model.Metrics{})).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/clusters/{cluster}/namespaces/{namespace}/ingresses/{ingress}").
		To(h.handleIngressMetricsQuery).
		Doc("Get Ingress-level metric data for the specific Ingress. Navigate to the Ingress by the Ingress's cluster and namespace.").
		Param(ws.PathParameter("cluster", "The name of cluster").DataType("string").Required(true)).
		Param(ws.PathParameter("namespace", "The name of the namespace.").DataType("string").Required(true)).
		Param(ws.PathParameter("ingress", "ingress name.").DataType("string").Required(true)).
		Param(ws.QueryParameter("job", "The job name filter. Ingress could be served by multi Ingress controllers, The job filters metric from a specific controller.").DataType("string").Required(true)).
		Param(ws.QueryParameter("pod", "The pod filter.").DataType("string")).
		Param(ws.QueryParameter("duration", "The duration is the time window of Range Vector. The format is [0-9]+[smhdwy]. Defaults to 5m (i.e. 5 min).").DataType("string").Required(false)).
		Param(ws.QueryParameter("metrics_filter", "The metric name filter consists of a regexp pattern. It specifies which metric data to return. For example, the following filter matches both PVC available and used inodes: `pvc_inodes_available|pvc_inodes_used`. View available metrics at [kubesphere.io](https://docs.kubesphere.io/advanced-v2.0/zh-CN/api-reference/monitoring-metrics/).").DataType("string").Required(false)).
		Param(ws.QueryParameter("start", "Start time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1559347200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("end", "End time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1561939200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("step", "Time interval. Retrieve metric data at a fixed interval within the time range of start and end. It requires both **start** and **end** are provided. The format is [0-9]+[smhdwy]. Defaults to 10m (i.e. 10 min).").DataType("string").DefaultValue("10m").Required(false)).
		Param(ws.QueryParameter("time", "A timestamp in Unix time format. Retrieve metric data at a single point in time. Defaults to now. Time and the combination of start, end, step are mutually exclusive.").DataType("string").Required(false)).
		Metadata(restfulspec.KeyOpenAPITags, []string{constants.IngressMetricsTag}).
		Writes(model.Metrics{}).
		Returns(http.StatusOK, respOK, model.Metrics{})).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/components/{component}").
		To(h.handleComponentMetricsQuery).
		Doc("Get component-level metric data for all cluster's system component.").
		Param(ws.PathParameter("component", "system component to monitor. One of etcd, apiserver, scheduler.").DataType("string").Required(true)).
		Param(ws.QueryParameter("metrics_filter", "The metric name filter consists of a regexp pattern. It specifies which metric data to return. For example, the following filter matches both etcd server list and total size of the underlying database: `etcd_server_list|etcd_mvcc_db_size`. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("cluster_resources_filter", "The cluster filter consists of a regexp pattern. It specifies which cluster data to return. For example, the following filter matches both cluster host and member-a: `host|member-a`.").DataType("string").Required(false)).
		Param(ws.QueryParameter("start", "Start time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1559347200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("end", "End time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1561939200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("step", "Time interval. Retrieve metric data at a fixed interval within the time range of start and end. It requires both **start** and **end** are provided. The format is [0-9]+[smhdwy]. Defaults to 10m (i.e. 10 min).").DataType("string").DefaultValue("10m").Required(false)).
		Param(ws.QueryParameter("time", "A timestamp in Unix time format. Retrieve metric data at a single point in time. Defaults to now. Time and the combination of start, end, step are mutually exclusive.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_metric", "Sort components by the specified metric. Not applicable if **start** and **end** are provided.").DataType("string").Required(false)).
		Param(ws.QueryParameter("sort_type", "Sort order. One of asc, desc.").DefaultValue("desc.").DataType("string").Required(false)).
		Param(ws.QueryParameter("page", "The page number. This field paginates result data of each metric, then returns a specific page. For example, setting **page** to 2 returns the second page. It only applies to sorted metric data.").DataType("integer").Required(false)).
		Param(ws.QueryParameter("limit", "Page size, the maximum number of results in a single page. Defaults to 5.").DataType("integer").Required(false).DefaultValue("5")).
		Param(ws.QueryParameter("type", "The query type. This field can be set to 'rank' for cluster ranking query or '' for others. Defaults to ''.").DataType("string").Required(false).DefaultValue("")).
		Metadata(restfulspec.KeyOpenAPITags, []string{constants.ComponentMetricsTag}).
		Writes(model.Metrics{}).
		Returns(http.StatusOK, respOK, model.Metrics{})).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/clusters/{cluster}/components/{component}").
		To(h.handleComponentMetricsQuery).
		Doc("Get component-level metrics for the specified system component of the specified cluster.").
		Param(ws.PathParameter("cluster", "The name of cluster").DataType("string").Required(true)).
		Param(ws.PathParameter("component", "system component to monitor. One of etcd, apiserver, scheduler.").DataType("string").Required(true)).
		Param(ws.QueryParameter("metrics_filter", "The metric name filter consists of a regexp pattern. It specifies which metric data to return. For example, the following filter matches both etcd server list and total size of the underlying database: `etcd_server_list|etcd_mvcc_db_size`. View available metrics at [kubesphere.io](https://docs.kubesphere.io/advanced-v2.0/zh-CN/api-reference/monitoring-metrics/).").DataType("string").Required(false)).
		Param(ws.QueryParameter("start", "Start time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1559347200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("end", "End time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1561939200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("step", "Time interval. Retrieve metric data at a fixed interval within the time range of start and end. It requires both **start** and **end** are provided. The format is [0-9]+[smhdwy]. Defaults to 10m (i.e. 10 min).").DataType("string").DefaultValue("10m").Required(false)).
		Param(ws.QueryParameter("time", "A timestamp in Unix time format. Retrieve metric data at a single point in time. Defaults to now. Time and the combination of start, end, step are mutually exclusive.").DataType("string").Required(false)).
		Metadata(restfulspec.KeyOpenAPITags, []string{constants.ComponentMetricsTag}).
		Writes(model.Metrics{}).
		Returns(http.StatusOK, respOK, model.Metrics{})).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/label/{label}/values").
		To(h.handleLabelValuesQuery).
		Doc("Get the values of the given label in all clusters.").
		Param(ws.PathParameter("label", "The name of the label. For example, query metric names: `__name__`").DataType("string").Required(true)).
		Param(ws.QueryParameter("cluster_resources_filter", "The cluster filter consists of a regexp pattern. It specifies which cluster data to return. For example, the following filter matches both cluster host and member-a: `host|member-a`.").DataType("string").Required(false)).
		Param(ws.QueryParameter("namespace_resources_filter", "The namespace filter consists of a regexp pattern. It specifies which namespace data to return. For example, the following filter matches both namespace test and default: `test|default`.").DataType("string").Required(false)).
		Param(ws.QueryParameter("start", "Start time of query. It is a string with Unix time format, eg. 1559347200. ").DataType("string").Required(true)).
		Param(ws.QueryParameter("end", "End time of query. It is a string with Unix time format, eg. 1561939200. ").DataType("string").Required(true)).
		Metadata(restfulspec.KeyOpenAPITags, []string{constants.CustomMetricsTag}).
		Writes(model.LabelValues{}).
		Returns(http.StatusOK, respOK, model.LabelValues{})).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/targets/metadata").
		To(h.handleMetadataQuery).
		Doc("Get metadata of metrics in all clusters.").
		Metadata(restfulspec.KeyOpenAPITags, []string{constants.CustomMetricsTag}).
		Writes(model.Metadata{}).
		Returns(http.StatusOK, respOK, model.Metadata{})).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/targets/labelsets").
		To(h.handleMetricLabelSetQuery).
		Doc("List all available labels and values of a metric within a specific time span in all clusters.").
		Param(ws.QueryParameter("metric", "The name of the metric").DataType("string").Required(true)).
		Param(ws.QueryParameter("start", "Start time of query. It is a string with Unix time format, eg. 1559347200. ").DataType("string").Required(true)).
		Param(ws.QueryParameter("end", "End time of query. It is a string with Unix time format, eg. 1561939200. ").DataType("string").Required(true)).
		Metadata(restfulspec.KeyOpenAPITags, []string{constants.CustomMetricsTag}).
		Writes(model.MetricLabelSet{}).
		Returns(http.StatusOK, respOK, model.MetricLabelSet{})).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/targets/query").
		To(h.handleAdhocQuery).
		Doc("Make an ad-hoc query in all clusters.").
		Param(ws.QueryParameter("expr", "The expression to be evaluated.").DataType("string").Required(false)).
		Param(ws.QueryParameter("start", "Start time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1559347200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("end", "End time of query. Use **start** and **end** to retrieve metric data over a time span. It is a string with Unix time format, eg. 1561939200. ").DataType("string").Required(false)).
		Param(ws.QueryParameter("step", "Time interval. Retrieve metric data at a fixed interval within the time range of start and end. It requires both **start** and **end** are provided. The format is [0-9]+[smhdwy]. Defaults to 10m (i.e. 10 min).").DataType("string").DefaultValue("10m").Required(false)).
		Param(ws.QueryParameter("time", "A timestamp in Unix time format. Retrieve metric data at a single point in time. Defaults to now. Time and the combination of start, end, step are mutually exclusive.").DataType("string").Required(false)).
		Metadata(restfulspec.KeyOpenAPITags, []string{constants.CustomMetricsTag}).
		Writes(monitoring.Metric{}).
		Returns(http.StatusOK, respOK, monitoring.Metric{})).
		Produces(restful.MIME_JSON)

	c.Add(ws)
	return nil
}
