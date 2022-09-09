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

package v1alpha1

import (
	"strconv"
	"time"

	"github.com/emicklei/go-restful"
	"github.com/pkg/errors"

	model "kubesphere.io/kubesphere/pkg/models/observability/monitoring"
	"kubesphere.io/kubesphere/pkg/simple/client/observability/monitoring"
)

const (
	DefaultStep   = 10 * time.Minute
	DefaultFilter = ".*"
	DefaultOrder  = model.OrderDescending
	DefaultPage   = 1
	DefaultLimit  = 5

	OperationQuery  = "query"
	OperationExport = "export"

	ComponentEtcd      = "etcd"
	ComponentAPIServer = "apiserver"
	ComponentScheduler = "scheduler"

	ErrNoHit             = "'end' or 'time' must be after the namespace creation time."
	ErrParamConflict     = "'time' and the combination of 'start' and 'end' are mutually exclusive."
	ErrInvalidStartEnd   = "'start' must be before 'end'."
	ErrInvalidPage       = "Invalid parameter 'page'."
	ErrInvalidLimit      = "Invalid parameter 'limit'."
	ErrParameterNotfound = "Parmameter [%s] not found"
)

type reqParams struct {
	time     string
	start    string
	end      string
	step     string
	duration string
	target   string
	order    string
	page     string
	limit    string

	queryType     string
	componentType string
	workloadKind  string

	metricFilter                  string
	resourceFilter                string
	clusterResourcesFilter        string
	workspaceResourcesFilter      string
	namespaceResourcesFilter      string
	podResourcesFilter            string
	namespacedNameResourcesFilter string
	storageclassResourcesFilter   string
	clusterName                   string
	nodeName                      string
	workspaceName                 string
	namespaceName                 string
	workloadName                  string
	podName                       string
	containerName                 string
	ingressName                   string
	storageClassName              string
	pvcName                       string
	job                           string

	label      string
	expression string
	metric     string
}

type queryOptions struct {
	metricFilter string
	namedMetrics []string

	start time.Time
	end   time.Time
	time  time.Time
	step  time.Duration

	target     string
	identifier string
	order      string
	page       int
	limit      int

	option monitoring.QueryOption
}

func (q queryOptions) isRangeQuery() bool {
	return q.time.IsZero()
}

func (q queryOptions) shouldSort() bool {
	return q.target != "" && q.identifier != ""
}

func parseRequestParams(req *restful.Request) reqParams {
	var r reqParams
	r.time = req.QueryParameter("time")
	r.start = req.QueryParameter("start")
	r.end = req.QueryParameter("end")
	r.step = req.QueryParameter("step")
	r.duration = req.QueryParameter("duration")
	r.target = req.QueryParameter("sort_metric")
	r.order = req.QueryParameter("sort_type")
	r.page = req.QueryParameter("page")
	r.limit = req.QueryParameter("limit")

	r.queryType = req.QueryParameter("type")
	r.workloadKind = req.PathParameter("kind")
	r.componentType = req.PathParameter("component")

	r.metricFilter = req.QueryParameter("metrics_filter")
	r.resourceFilter = req.QueryParameter("resources_filter")
	r.clusterResourcesFilter = req.QueryParameter("cluster_resources_filter")
	r.workspaceResourcesFilter = req.QueryParameter("workspace_resources_filter")
	r.namespaceResourcesFilter = req.QueryParameter("namespace_resources_filter")
	r.podResourcesFilter = req.QueryParameter("pod_resources_filter")
	r.namespacedNameResourcesFilter = req.QueryParameter("namespaced_resources_filter")
	r.storageclassResourcesFilter = req.QueryParameter("storageclass_resources_filter")
	r.clusterName = req.PathParameter("cluster")
	r.nodeName = req.PathParameter("node")
	r.workspaceName = req.PathParameter("workspace")
	r.namespaceName = req.PathParameter("namespace")
	r.workloadName = req.PathParameter("workload")
	r.podName = req.PathParameter("pod")
	//will be overide if "pod" in the path parameter. It is only used to filter ingress
	if r.podName == "" {
		r.podName = req.QueryParameter("pod")
	}
	r.containerName = req.PathParameter("container")
	r.ingressName = req.PathParameter("ingress")
	r.storageClassName = req.PathParameter("storageclass")
	r.pvcName = req.PathParameter("pvc")
	r.job = req.QueryParameter("job")

	r.label = req.PathParameter("label")
	r.expression = req.QueryParameter("expr")
	r.metric = req.QueryParameter("metric")

	return r
}

func (h handler) makeQueryOptions(r reqParams, lvl monitoring.Level) (q queryOptions, err error) {
	if r.resourceFilter == "" {
		r.resourceFilter = DefaultFilter
	}

	q.metricFilter = r.metricFilter
	if r.metricFilter == "" {
		q.metricFilter = DefaultFilter
	}

	switch lvl {
	case monitoring.LevelCluster:
		q.identifier = model.IdentifierCluster
		q.option = monitoring.SampleOption{
			ClusterName:            r.clusterName,
			ClusterResourcesFilter: r.resourceFilter,
			QueryType:              r.queryType,
			Level:                  monitoring.LevelCluster,
		}
		q.namedMetrics = model.ClusterMetrics

	case monitoring.LevelNode:
		q.identifier = model.IdentifierNode
		q.option = monitoring.SampleOption{
			Level:                  monitoring.LevelNode,
			QueryType:              r.queryType,
			ClusterName:            r.clusterName,
			ClusterResourcesFilter: r.clusterResourcesFilter,
			NodeName:               r.nodeName,
			NodeResourcesFilter:    r.resourceFilter,
		}
		q.namedMetrics = model.NodeMetrics

	case monitoring.LevelWorkspace:
		q.identifier = model.IdentifierWorkspace
		q.option = monitoring.SampleOption{
			Level:                    monitoring.LevelWorkspace,
			QueryType:                r.queryType,
			ClusterName:              r.clusterName,
			ClusterResourcesFilter:   r.clusterResourcesFilter,
			WorkspaceName:            r.workspaceName,
			WorkspaceResourcesFilter: r.resourceFilter,
		}
		q.namedMetrics = model.WorkspaceMetrics

	case monitoring.LevelNamespace:
		q.identifier = model.IdentifierNamespace
		q.option = monitoring.SampleOption{
			Level:                    monitoring.LevelNamespace,
			QueryType:                r.queryType,
			ClusterName:              r.clusterName,
			ClusterResourcesFilter:   r.clusterResourcesFilter,
			WorkspaceName:            r.workspaceName,
			WorkspaceResourcesFilter: r.workspaceResourcesFilter,
			NamespaceName:            r.namespaceName,
			NamespaceResourcesFilter: r.resourceFilter,
		}
		q.namedMetrics = model.NamespaceMetrics

	case monitoring.LevelWorkload:
		q.identifier = model.IdentifierWorkload
		q.option = monitoring.SampleOption{
			Level:                    monitoring.LevelWorkload,
			QueryType:                r.queryType,
			WorkloadKind:             r.workloadKind,
			ClusterName:              r.clusterName,
			ClusterResourcesFilter:   r.clusterResourcesFilter,
			NamespaceResourcesFilter: r.namespaceResourcesFilter,
			NamespaceName:            r.namespaceName,
			WorkloadResourcesFilter:  r.resourceFilter,
		}
		q.namedMetrics = model.WorkloadMetrics

	case monitoring.LevelPod:
		q.identifier = model.IdentifierPod
		q.option = monitoring.SampleOption{
			Level:                    monitoring.LevelPod,
			QueryType:                r.queryType,
			ClusterName:              r.clusterName,
			ClusterResourcesFilter:   r.clusterResourcesFilter,
			NodeName:                 r.nodeName,
			NamespaceName:            r.namespaceName,
			NamespaceResourcesFilter: r.namespaceResourcesFilter,
			WorkloadKind:             r.workloadKind,
			WorkloadName:             r.workloadName,
			PodName:                  r.podName,
			PodResourcesFilter:       r.resourceFilter,
		}
		q.namedMetrics = model.PodMetrics

	case monitoring.LevelContainer:
		q.identifier = model.IdentifierContainer
		q.option = monitoring.SampleOption{
			Level:                    monitoring.LevelContainer,
			QueryType:                r.queryType,
			ClusterName:              r.clusterName,
			ClusterResourcesFilter:   r.clusterResourcesFilter,
			NamespaceName:            r.namespaceName,
			NamespaceResourcesFilter: r.namespaceResourcesFilter,
			PodName:                  r.podName,
			PodResourcesFilter:       r.podResourcesFilter,
			ContainerName:            r.containerName,
			ContainerResourcesFilter: r.resourceFilter,
		}
		q.namedMetrics = model.ContainerMetrics

	case monitoring.LevelPVC:
		q.identifier = model.IdentifierPVC
		q.option = monitoring.SampleOption{
			Level:                       monitoring.LevelPVC,
			QueryType:                   r.queryType,
			ClusterName:                 r.clusterName,
			ClusterResourcesFilter:      r.clusterResourcesFilter,
			NamespaceName:               r.namespaceName,
			NamespaceResourcesFilter:    r.namespaceResourcesFilter,
			StorageClassName:            r.storageClassName,
			StorageClassResourcesFilter: r.storageclassResourcesFilter,
			PVCName:                     r.pvcName,
			PVCResourcesFilter:          r.resourceFilter,
		}
		q.namedMetrics = model.PVCMetrics

	case monitoring.LevelIngress:
		q.identifier = model.IdentifierIngress
		var du *time.Duration
		// duration param is used in none Range Query to pass vector's time duration.
		if r.time != "" {
			s, err := time.ParseDuration(r.duration)
			if err == nil {
				du = &s
			}
		}
		q.option = monitoring.SampleOption{
			Level:                    monitoring.LevelIngress,
			QueryType:                r.queryType,
			ClusterName:              r.clusterName,
			ClusterResourcesFilter:   r.clusterResourcesFilter,
			NamespaceName:            r.namespaceName,
			NamespaceResourcesFilter: r.namespaceResourcesFilter,
			IngressName:              r.ingressName,
			IngressResourcesFilter:   r.resourceFilter,
			Job:                      r.job,
			PodName:                  r.podName,
			Duration:                 du,
		}
		q.namedMetrics = model.IngressMetrics

	case monitoring.LevelComponent:
		q.identifier = model.IdentifierComponent
		q.option = monitoring.SampleOption{
			Level:                  monitoring.LevelComponent,
			QueryType:              r.queryType,
			ContainerName:          r.clusterName,
			ClusterResourcesFilter: r.clusterResourcesFilter,
		}
		switch r.componentType {
		case ComponentEtcd:
			q.namedMetrics = model.EtcdMetrics
		case ComponentAPIServer:
			q.namedMetrics = model.APIServerMetrics
		case ComponentScheduler:
			q.namedMetrics = model.SchedulerMetrics
		}
	}

	// Parse time params
	if r.start != "" && r.end != "" {
		startInt, err := strconv.ParseInt(r.start, 10, 64)
		if err != nil {
			return q, err
		}
		q.start = time.Unix(startInt, 0)

		endInt, err := strconv.ParseInt(r.end, 10, 64)
		if err != nil {
			return q, err
		}
		q.end = time.Unix(endInt, 0)

		if r.step == "" {
			q.step = DefaultStep
		} else {
			q.step, err = time.ParseDuration(r.step)
			if err != nil {
				return q, err
			}
		}

		if q.start.After(q.end) {
			return q, errors.New(ErrInvalidStartEnd)
		}
	} else if r.start == "" && r.end == "" {
		if r.time == "" {
			q.time = time.Now()
		} else {
			timeInt, err := strconv.ParseInt(r.time, 10, 64)
			if err != nil {
				return q, err
			}
			q.time = time.Unix(timeInt, 0)
		}
	} else {
		return q, errors.Errorf(ErrParamConflict)
	}

	// Parse sorting and paging params
	if r.target != "" {
		q.target = r.target
		q.page = DefaultPage
		q.limit = DefaultLimit
		q.order = r.order
		if r.order != model.OrderAscending {
			q.order = DefaultOrder
		}
		if r.page != "" {
			q.page, err = strconv.Atoi(r.page)
			if err != nil || q.page <= 0 {
				return q, errors.New(ErrInvalidPage)
			}
		}
		if r.limit != "" {
			q.limit, err = strconv.Atoi(r.limit)
			if err != nil || q.limit <= 0 {
				return q, errors.New(ErrInvalidLimit)
			}
		}
	}

	return q, nil
}
