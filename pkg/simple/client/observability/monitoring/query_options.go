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

package monitoring

import (
	"time"
)

type Level int

const (
	LevelCluster = 1 << iota
	LevelNode
	LevelWorkspace
	LevelNamespace
	LevelWorkload
	LevelService
	LevelPod
	LevelContainer
	LevelPVC
	LevelIngress
	LevelComponent
)

type QueryOption interface {
	Apply(*QueryOptions)
}

type QueryOptions struct {
	Level     Level
	QueryType string

	ClusterName                   string
	ClusterResourcesFilter        string
	NodeName                      string
	NodeResourcesFilter           string
	WorkspaceName                 string
	WorkspaceResourcesFilter      string
	NamespaceName                 string
	NamespaceResourcesFilter      string
	WorkloadKind                  string
	WorkloadName                  string
	WorkloadResourcesFilter       string
	PodName                       string
	PodResourcesFilter            string
	NamespacedNameResourcesFilter string
	ContainerName                 string
	ContainerResourcesFilter      string
	IngressName                   string
	IngressResourcesFilter        string
	Job                           string
	Duration                      *time.Duration
	StorageClassName              string
	StorageClassResourcesFilter   string
	PVCName                       string
	PVCResourcesFilter            string
}

func NewQueryOptions() *QueryOptions {
	return &QueryOptions{}
}

type SampleOption struct {
	Level     Level
	QueryType string

	ClusterName                   string
	ClusterResourcesFilter        string
	NodeName                      string
	NodeResourcesFilter           string
	WorkspaceName                 string
	WorkspaceResourcesFilter      string
	NamespaceName                 string
	NamespaceResourcesFilter      string
	WorkloadKind                  string
	WorkloadName                  string
	WorkloadResourcesFilter       string
	PodName                       string
	PodResourcesFilter            string
	NamespacedNameResourcesFilter string
	ContainerName                 string
	ContainerResourcesFilter      string
	IngressName                   string
	IngressResourcesFilter        string
	Job                           string
	Duration                      *time.Duration
	StorageClassName              string
	StorageClassResourcesFilter   string
	PVCName                       string
	PVCResourcesFilter            string
}

func (so SampleOption) Apply(o *QueryOptions) {
	o.Level = so.Level
	o.QueryType = so.QueryType

	o.ClusterName = so.ClusterName
	o.ClusterResourcesFilter = so.ClusterResourcesFilter
	o.NodeName = so.NodeName
	o.NodeResourcesFilter = so.NodeResourcesFilter
	o.WorkspaceName = so.WorkspaceName
	o.WorkspaceResourcesFilter = so.WorkspaceResourcesFilter
	o.WorkloadName = so.WorkloadName
	o.WorkloadKind = so.WorkloadName
	o.WorkloadResourcesFilter = so.WorkloadResourcesFilter
	o.NamespaceName = so.NamespaceName
	o.NamespaceResourcesFilter = so.NamespaceResourcesFilter
	o.PodName = so.PodName
	o.PodResourcesFilter = so.PodResourcesFilter
	o.NamespacedNameResourcesFilter = so.NamespacedNameResourcesFilter
	o.ContainerName = so.ContainerName
	o.ContainerResourcesFilter = so.ContainerResourcesFilter
	o.IngressName = so.IngressName
	o.IngressResourcesFilter = so.IngressResourcesFilter
	o.Job = so.Job
	o.Duration = so.Duration
	o.StorageClassName = so.StorageClassName
	o.StorageClassResourcesFilter = so.StorageClassResourcesFilter
	o.PVCName = so.PVCName
	o.PVCResourcesFilter = so.PVCResourcesFilter
}
