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

package v2beta2

import (
	"net/http"

	"github.com/emicklei/go-restful"
	openapi "github.com/emicklei/go-restful-openapi"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"

	"kubesphere.io/kubesphere/pkg/api"
	notificationv2beta2 "kubesphere.io/kubesphere/pkg/api/notification/v2beta2"
	"kubesphere.io/kubesphere/pkg/apiserver/runtime"
	kubesphere "kubesphere.io/kubesphere/pkg/client/clientset/versioned"
	"kubesphere.io/kubesphere/pkg/constants"
	"kubesphere.io/kubesphere/pkg/informers"
	nm "kubesphere.io/kubesphere/pkg/simple/client/notification"
	notificationclient "kubesphere.io/kubesphere/pkg/simple/client/notification"
)

const (
	KeyOpenAPITags = openapi.KeyOpenAPITags
)

var GroupVersion = schema.GroupVersion{Group: "notification.kubesphere.io", Version: "v2beta2"}

func AddToContainer(
	container *restful.Container,
	informers informers.InformerFactory,
	k8sClient kubernetes.Interface,
	ksClient kubesphere.Interface,
	notificationClient notificationclient.Client,
	option *nm.Options) error {
	h := newNotificationHandler(informers, k8sClient, ksClient, notificationClient, option)
	ws := runtime.NewWebService(GroupVersion)
	ws.Route(ws.POST("/verification").
		Reads("").
		To(h.Verify).
		Returns(http.StatusOK, api.StatusOK, http.Response{}.Body)).
		Doc("Provide validation for notification-manager information")
	ws.Route(ws.POST("/users/{user}/verification").
		To(h.Verify).
		Param(ws.PathParameter("user", "user name")).
		Returns(http.StatusOK, api.StatusOK, http.Response{}.Body)).
		Doc("Provide validation for notification-manager information")

	ws.Route(ws.GET("/notifications/search").
		To(h.SearchNotification).
		Doc("Query notification history against the cluster").
		Param(ws.QueryParameter("operation", "Operation type. This can be one of these types: query (for querying logs) and export (for exporting logs). Defaults to query.").DefaultValue("query").DataType("string").Required(false)).
		Param(ws.QueryParameter("alertname", "A comma-separated list of alert name.")).
		Param(ws.QueryParameter("alertname_fuzzy", "A comma-separated list of fuzzy alert name.")).
		Param(ws.QueryParameter("alerttype", "A comma-separated list of alert type.")).
		Param(ws.QueryParameter("alerttype_fuzzy", "A comma-separated list of fuzzy alert type.")).
		Param(ws.QueryParameter("severity", "A comma-separated list of severity.")).
		Param(ws.QueryParameter("severity_fuzzy", "A comma-separated list of fuzzy severity.")).
		Param(ws.QueryParameter("namespace", "A comma-separated list of namespaces.")).
		Param(ws.QueryParameter("namespace_fuzzy", "A comma-separated list of fuzzy namespaces.")).
		Param(ws.QueryParameter("service", "A comma-separated list of service name.")).
		Param(ws.QueryParameter("service_fuzzy", "A comma-separated list of fuzzy service name.")).
		Param(ws.QueryParameter("pod", "A comma-separated list of pod name.")).
		Param(ws.QueryParameter("pod_fuzzy", "A comma-separated list of fuzzy pod name.")).
		Param(ws.QueryParameter("container", "A comma-separated list of container name.")).
		Param(ws.QueryParameter("container_fuzzy", "A comma-separated list of fuzzy container name.")).
		Param(ws.QueryParameter("message_fuzzy", "Alert message.")).
		Param(ws.QueryParameter("start_time", "Start time of query (limits `NotificationTime`). The format is a string representing seconds since the epoch, eg. 1136214245.")).
		Param(ws.QueryParameter("end_time", "End time of query (limits `NotificationTime`). The format is a string representing seconds since the epoch, eg. 1136214245.")).
		Param(ws.QueryParameter("sort", "Sort field.").DataType("string")).
		Param(ws.QueryParameter("order", "Sort order. One of asc, desc.").DataType("string").DefaultValue("desc")).
		Param(ws.QueryParameter("from", "The offset from the result set. This field returns query results from the specified offset.").DataType("integer").DefaultValue("0").Required(false)).
		Param(ws.QueryParameter("size", "Size of result set to return. Defaults to 10 (i.e. 10 event records).").DataType("integer").DefaultValue("10").Required(false)).
		Metadata(KeyOpenAPITags, []string{constants.NotificationQueryTag}).
		Writes(notificationv2beta2.APIResponse{}).
		Returns(http.StatusOK, api.StatusOK, notificationv2beta2.APIResponse{}))

	container.Add(ws)
	return nil
}
