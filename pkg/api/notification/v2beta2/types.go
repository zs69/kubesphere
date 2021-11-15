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

package v1alpha2

import (
	"strconv"
	"time"

	"github.com/emicklei/go-restful"

	"kubesphere.io/kubesphere/pkg/simple/client/notification"
)

type APIResponse struct {
	*notification.Notifications `json:",inline" description:"query results"`
}

type Query struct {
	AlertName      string `json:"alertname,omitempty"`
	AlertNameFuzzy string `json:"alertname_fuzzy,omitempty"`
	AlertType      string `json:"alerttype,omitempty"`
	AlertTypeFuzzy string `json:"alerttype_fuzzy,omitempty"`
	Severity       string `json:"severity,omitempty"`
	SeverityFuzzy  string `json:"severity_fuzzy,omitempty"`
	Namespace      string `json:"namespace,omitempty"`
	NamespaceFuzzy string `json:"namespace_fuzzy,omitempty"`
	Service        string `json:"service,omitempty"`
	ServiceFuzzy   string `json:"service_fuzzy,omitempty"`
	Container      string `json:"container,omitempty"`
	ContainerFuzzy string `json:"container_fuzzy,omitempty"`
	Pod            string `json:"pod,omitempty"`
	PodFuzzy       string `json:"pod_fuzzy,omitempty"`
	Message        string `json:"message,omitempty"`

	StartTime time.Time `json:"start_time,omitempty"`
	EndTime   time.Time `json:"end_time,omitempty"`

	Sort  string `json:"sort,omitempty"`
	Order string `json:"order,omitempty"`
	From  int64  `json:"from,omitempty"`
	Size  int64  `json:"size,omitempty"`
}

func ParseQueryParameter(req *restful.Request) (*Query, error) {
	q := &Query{}

	q.AlertName = req.QueryParameter("alertname")
	q.AlertNameFuzzy = req.QueryParameter("alertname_fuzzy")
	q.AlertType = req.QueryParameter("alerttype")
	q.AlertTypeFuzzy = req.QueryParameter("alerttype_fuzzy")
	q.Severity = req.QueryParameter("severity")
	q.SeverityFuzzy = req.QueryParameter("severity_fuzzy")
	q.Namespace = req.QueryParameter("namespace")
	q.NamespaceFuzzy = req.QueryParameter("namespace_fuzzy")
	q.Service = req.QueryParameter("service")
	q.ServiceFuzzy = req.QueryParameter("service_fuzzy")
	q.Container = req.QueryParameter("container")
	q.ContainerFuzzy = req.QueryParameter("container_fuzzy")
	q.Pod = req.QueryParameter("pod")
	q.PodFuzzy = req.QueryParameter("pod_fuzzy")
	q.Message = req.QueryParameter("message_fuzzy")

	if str := req.QueryParameter("start_time"); str != "" {
		sec, err := strconv.ParseInt(str, 10, 64)
		if err != nil {
			return nil, err
		}
		t := time.Unix(sec, 0)
		q.StartTime = t
	}
	if str := req.QueryParameter("end_time"); str != "" {
		sec, err := strconv.ParseInt(str, 10, 64)
		if err != nil {
			return nil, err
		}
		t := time.Unix(sec, 0)
		q.EndTime = t
	}

	q.From, _ = strconv.ParseInt(req.QueryParameter("from"), 10, 64)
	size, err := strconv.ParseInt(req.QueryParameter("size"), 10, 64)
	if err != nil {
		size = 10
	}
	q.Size = size

	q.Sort = req.QueryParameter("sort")
	if q.Order = req.QueryParameter("order"); q.Order != "asc" {
		q.Order = "desc"
	}
	return q, nil
}
