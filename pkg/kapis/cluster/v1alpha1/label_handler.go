/*
Copyright 2022 KubeSphere Authors

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
	"context"
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/sets"
	clusterv1alpha1 "kubesphere.io/api/cluster/v1alpha1"

	"kubesphere.io/kubesphere/pkg/api"
	apiv1alpha1 "kubesphere.io/kubesphere/pkg/api/cluster/v1alpha1"
)

func labelExists(req apiv1alpha1.CreateLabelRequest, labels []*clusterv1alpha1.Label) bool {
	for _, label := range labels {
		if label.Spec.Key == req.Key && label.Spec.Value == req.Value {
			return true
		}
	}
	return false
}

func (h *handler) createLabels(request *restful.Request, response *restful.Response) {
	var labelRequests []apiv1alpha1.CreateLabelRequest
	if err := request.ReadEntity(&labelRequests); err != nil {
		api.HandleBadRequest(response, request, err)
		return
	}

	allLabels, err := h.labelLister.List(labels.Everything())
	if err != nil {
		api.HandleBadRequest(response, request, err)
		return
	}

	results := make([]*clusterv1alpha1.Label, 0)
	for _, r := range labelRequests {
		if labelExists(r, allLabels) {
			api.HandleBadRequest(response, request, fmt.Errorf("label %s/%s already exists", r.Key, r.Value))
			return
		}

		obj := &clusterv1alpha1.Label{
			ObjectMeta: metav1.ObjectMeta{
				Name:       rand.String(6),
				Finalizers: []string{clusterv1alpha1.LabelFinalizer},
			},
			Spec: clusterv1alpha1.LabelSpec{
				Key:   r.Key,
				Value: r.Value,
			},
		}
		created, err := h.ksclient.ClusterV1alpha1().Labels().Create(context.TODO(), obj, metav1.CreateOptions{})
		if err != nil {
			api.HandleBadRequest(response, request, err)
			return
		}
		results = append(results, created)
	}
	response.WriteEntity(results)
}

func (h *handler) updateLabel(request *restful.Request, response *restful.Response) {
	label, err := h.labelLister.Get(request.PathParameter("label"))
	if err != nil {
		api.HandleBadRequest(response, request, err)
		return
	}
	label = label.DeepCopy()

	switch request.QueryParameter("action") {
	case "unbind": // unbind clusters
		var unbindRequest apiv1alpha1.UnbindClustersRequest
		if err = request.ReadEntity(&unbindRequest); err != nil {
			api.HandleBadRequest(response, request, err)
			return
		}
		for _, name := range unbindRequest.Clusters {
			cluster, err := h.clusterLister.Get(name)
			if err != nil {
				api.HandleBadRequest(response, request, err)
				return
			}
			cluster = cluster.DeepCopy()
			delete(cluster.Labels, fmt.Sprintf(clusterv1alpha1.ClusterLabelFormat, label.Name))
			if _, err = h.ksclient.ClusterV1alpha1().Clusters().Update(context.TODO(), cluster, metav1.UpdateOptions{}); err != nil {
				api.HandleBadRequest(response, request, err)
				return
			}
		}
		clusters := sets.NewString(label.Spec.Clusters...)
		clusters.Delete(unbindRequest.Clusters...)
		label.Spec.Clusters = clusters.List()
		updated, err := h.ksclient.ClusterV1alpha1().Labels().Update(context.TODO(), label, metav1.UpdateOptions{})
		if err != nil {
			api.HandleBadRequest(response, request, err)
			return
		}
		response.WriteEntity(updated)
	default: // update label key/value
		var labelRequest apiv1alpha1.CreateLabelRequest
		if err = request.ReadEntity(&labelRequest); err != nil {
			api.HandleBadRequest(response, request, err)
			return
		}

		allLabels, err := h.labelLister.List(labels.Everything())
		if err != nil {
			api.HandleBadRequest(response, request, err)
			return
		}

		if labelExists(labelRequest, allLabels) {
			api.HandleBadRequest(response, request, fmt.Errorf("label %s/%s already exists", labelRequest.Key, labelRequest.Value))
			return
		}
		label.Spec.Key = labelRequest.Key
		label.Spec.Value = labelRequest.Value
		updated, err := h.ksclient.ClusterV1alpha1().Labels().Update(context.TODO(), label, metav1.UpdateOptions{})
		if err != nil {
			api.HandleBadRequest(response, request, err)
			return
		}
		response.WriteEntity(updated)
	}
}

func (h *handler) deleteLabels(request *restful.Request, response *restful.Response) {
	var names []string
	if err := request.ReadEntity(&names); err != nil {
		api.HandleBadRequest(response, request, err)
		return
	}

	for _, name := range names {
		if err := h.ksclient.ClusterV1alpha1().Labels().Delete(context.TODO(), name, metav1.DeleteOptions{}); err != nil {
			api.HandleBadRequest(response, request, err)
			return
		}
	}
	response.WriteHeader(http.StatusOK)
}

func (h *handler) bindingClusters(request *restful.Request, response *restful.Response) {
	var bindingRequest apiv1alpha1.BindingClustersRequest
	if err := request.ReadEntity(&bindingRequest); err != nil {
		api.HandleBadRequest(response, request, err)
		return
	}

	for _, name := range bindingRequest.Labels {
		label, err := h.labelLister.Get(name)
		if err != nil {
			api.HandleBadRequest(response, request, err)
			return
		}
		label = label.DeepCopy()
		label.Spec.Clusters = append(label.Spec.Clusters, bindingRequest.Clusters...)
		if _, err = h.ksclient.ClusterV1alpha1().Labels().Update(context.TODO(), label, metav1.UpdateOptions{}); err != nil {
			api.HandleBadRequest(response, request, err)
			return
		}
	}

	response.WriteHeader(http.StatusOK)
}

func (h *handler) listLabelGroups(request *restful.Request, response *restful.Response) {
	allLabels, err := h.labelLister.List(labels.Everything())
	if err != nil {
		api.HandleBadRequest(response, request, err)
		return
	}

	results := make(map[string][]apiv1alpha1.LabelValue)
	for _, label := range allLabels {
		results[label.Spec.Key] = append(results[label.Spec.Key], apiv1alpha1.LabelValue{
			Value: label.Spec.Value,
			ID:    label.Name,
		})
	}

	response.WriteEntity(results)
}
