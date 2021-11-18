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

package clusterinfo

import (
	"context"
	"sync/atomic"
	"time"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	teypedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/klog"

	clusterinformer "kubesphere.io/kubesphere/pkg/client/informers/externalversions/cluster/v1alpha1"
	"kubesphere.io/kubesphere/pkg/utils/clusterclient"
)

type ClusterNodeEvent struct{}

type ClusterNode struct {
	// If cluster is empty, then this is an event from current cluster.
	Cluster string
	Node    *corev1.Node
}

type clusterInfo map[string]map[string]*corev1.Node

type ClusterInfoManager struct {
	lastNodesMap atomic.Value
	nodesMap     atomic.Value

	clusterClient clusterclient.ClusterClients
	// If a node is updated, send an event to this chan.
	UpdateChan chan<- *ClusterNodeEvent
}

// NewClusterInfoManager creates a manger to store node's cache.
// It will update the cache periodically.
func NewClusterInfoManager(clusterInformer clusterinformer.ClusterInformer, c chan<- *ClusterNodeEvent) *ClusterInfoManager {
	return &ClusterInfoManager{
		clusterClient: clusterclient.NewClusterClient(clusterInformer),
		UpdateChan:    c,
	}
}

// GetClusterInfo get all the nodes of all the clusters from the local cache and the number of the cluster.
func (cim *ClusterInfoManager) GetClusterInfo() ([]ClusterNode, int, error) {
	// Get the node info from cache as fetching all the nodes is time-consuming.
	val := cim.nodesMap.Load()
	var clusters clusterInfo
	if val != nil {
		clusters = val.(clusterInfo)
	}

	var clusterNodes []ClusterNode
	for cluster, nodes := range clusters {
		for i := range nodes {
			cn := ClusterNode{
				Node:    nodes[i],
				Cluster: cluster,
			}
			clusterNodes = append(clusterNodes, cn)
		}
	}

	return clusterNodes, len(clusters), nil
}

func (cim *ClusterInfoManager) Run(stop <-chan struct{}) {
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ticker.C:
			// Fetch nodes info from all member clusters and the host cluster periodically.
			klog.V(2).Infof("start to fetch nodes info")
			allClusters := cim.clusterClient.Clusters()
			nodesMap := make(clusterInfo)
			for name, cluster := range allClusters {
				if cim.clusterClient.IsClusterReady(cluster) {
					restConfig := cim.clusterClient.GetRestConfig(name)
					nodeClient, err := teypedcorev1.NewForConfig(restConfig)
					if err != nil {
						klog.Errorf("create reset client for cluster: %s failed, error: %s", name, err)
						continue
					}
					nodeList, err := nodeClient.Nodes().List(context.Background(), v1.ListOptions{})

					if err != nil {
						klog.Errorf("list node from cluster: %s, error: %s", name, err)
					}

					clusterName := cluster.Name
					// Set cluster name to empty to differentiate it from the member clusters.
					if cim.clusterClient.IsHostCluster(cluster) {
						clusterName = ""
					}

					nodesMap[clusterName] = make(map[string]*corev1.Node, len(nodeList.Items))

					for i := range nodeList.Items {
						nodesMap[clusterName][nodeList.Items[i].Name] = &nodeList.Items[i]
					}
				}
				klog.V(2).Infof("cluster not ready: %s", name)
			}

			currentNodesMap := cim.nodesMap.Load()
			if currentNodesMap == nil {
				currentNodesMap = make(clusterInfo)
			}

			cim.lastNodesMap.Store(currentNodesMap)
			// Get the node info from cache as fetching all the nodes is time-consuming.
			cim.nodesMap.Store(nodesMap)
			cim.events()
			klog.V(2).Infof("fetch nodes info end")

			ticker.Reset(30 * time.Second)
		case <-stop:
			klog.Infof("cluster info manager stop")
			return
		}
	}
}

// updated checks whether cluster changed or not.
func updated(newNodesMap, oldNodesMap clusterInfo) bool {
	for cluster := range newNodesMap {
		if oldNodes, exists := oldNodesMap[cluster]; exists {
			for nodeName, newNode := range newNodesMap[cluster] {
				if oldNode, exists := oldNodes[nodeName]; !exists {
					// New Node added.
					return true
				} else if isNodeChanged(oldNode, newNode) {
					// Node changed.
					return true
				}
			}
		} else {
			// New cluster added.
			return true
		}
	}

	for cluster := range oldNodesMap {
		if newNodes, exists := newNodesMap[cluster]; exists {
			for nodeName, oldNode := range oldNodesMap[cluster] {
				if newNode, exists := newNodes[nodeName]; !exists {
					// Node deleted.
					return true
				} else if isNodeChanged(oldNode, newNode) {
					// Node changed.
					return true
				}
			}
		} else {
			// Cluster deleted.
			return true
		}
	}

	return false
}

func isNodeChanged(node1, node2 *corev1.Node) bool {
	if node1.Spec.Unschedulable != node2.Spec.Unschedulable {
		return true
	}

	core1 := node1.Status.Capacity.Cpu()
	core2 := node2.Status.Capacity.Cpu()
	// CPU core num changed.
	if core1 != nil && core2 != nil {
		if !core1.Equal(*core2) {
			return true
		}
	} else {
		if (core1 == nil && core2 != nil) && (core1 != nil && core2 == nil) {
			return true
		}
	}

	return false
}

// events sends an event to cim.UpdateChan if node info changed.
// Then the licence controller will update the license state.
func (cim *ClusterInfoManager) events() {
	if cim.UpdateChan == nil {
		return
	}
	v := cim.lastNodesMap.Load()
	lastNodesMap := v.(clusterInfo)
	v = cim.nodesMap.Load()
	newNodesMap := v.(clusterInfo)
	// If node has been updated, send this event.
	if updated(newNodesMap, lastNodesMap) {
		cim.UpdateChan <- &ClusterNodeEvent{}
	}
}
