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

	openpitrixoptions "kubesphere.io/kubesphere/pkg/simple/client/openpitrix"

	restful "github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"helm.sh/helm/v3/pkg/release"
	"k8s.io/apimachinery/pkg/runtime/schema"
	v1 "k8s.io/client-go/informers/core/v1"

	"kubesphere.io/kubesphere/pkg/api"
	"kubesphere.io/kubesphere/pkg/apiserver/query"
	"kubesphere.io/kubesphere/pkg/client/clientset/versioned"
	"kubesphere.io/kubesphere/pkg/client/informers/externalversions"
	"kubesphere.io/kubesphere/pkg/constants"
	"kubesphere.io/kubesphere/pkg/models"
	"kubesphere.io/kubesphere/pkg/models/openpitrix"
	"kubesphere.io/kubesphere/pkg/server/errors"
	"kubesphere.io/kubesphere/pkg/server/params"
	"kubesphere.io/kubesphere/pkg/utils/clusterclient"

	"kubesphere.io/kubesphere/pkg/apiserver/runtime"
)

const (
	GroupName = "native.helm"
	Resource  = "release"
)

var GroupVersion = schema.GroupVersion{Group: GroupName, Version: "v1alpha1"}

func AddToContainer(container *restful.Container, ksFactory externalversions.SharedInformerFactory,
	ksClient versioned.Interface, cc clusterclient.ClusterClients,
	secretInformer v1.SecretInformer, configmapInformer v1.ConfigMapInformer, option *openpitrixoptions.Options) error {

	if option == nil || option.NativeHelmReleaseOptions == nil || option.NativeHelmReleaseOptions.Enable == false {
		return nil
	}

	webservice := runtime.NewWebService(GroupVersion)
	handler := newHelmShReleaseHandler(ksFactory, ksClient, cc, secretInformer, configmapInformer)

	webservice.Route(webservice.GET("/namespaces/{namespace}/releases/{name}").
		To(handler.Get).
		Doc("Get the helm release info from cluster.").
		Param(webservice.PathParameter("namespace", "The namespace of the release")).
		Param(webservice.PathParameter("name", "The name of the release")).
		Returns(http.StatusOK, api.StatusOK, release.Release{}))

	webservice.Route(webservice.GET("/namespaces/{namespace}/releases").
		To(handler.ListReleases).
		Doc("List the helm release in the namespace").
		Param(webservice.PathParameter("namespace", "The namespace of the release")).
		Param(webservice.QueryParameter(query.ParameterName, "name used to do filtering").Required(false)).
		Param(webservice.QueryParameter(query.ParameterPage, "page").Required(false).DataFormat("page=%d").DefaultValue("page=1")).
		Param(webservice.QueryParameter(query.ParameterLimit, "limit").Required(false)).
		Param(webservice.QueryParameter(query.ParameterAscending, "sort parameters, e.g. reverse=true").Required(false).DefaultValue("ascending=false")).
		Param(webservice.QueryParameter(query.ParameterOrderBy, "sort parameters, e.g. orderBy=createTime")).
		Returns(http.StatusOK, api.StatusOK, models.PageableResponse{}))

	webservice.Route(webservice.GET("/namespaces/{namespace}/applications/{application}").
		To(handler.DescribeApplication).
		Returns(http.StatusOK, api.StatusOK, openpitrix.Application{}).
		Metadata(restfulspec.KeyOpenAPITags, []string{constants.NamespaceResourcesTag}).
		Doc("Describe the specified application of the namespace").
		Param(webservice.PathParameter("namespace", "the name of the project").Required(true)).
		Param(webservice.PathParameter("application", "the id of the application").Required(true)))

	webservice.Route(webservice.GET("/namespaces/{namespace}/applications").
		To(handler.ListApplications).
		Returns(http.StatusOK, api.StatusOK, models.PageableResponse{}).
		Metadata(restfulspec.KeyOpenAPITags, []string{constants.NamespaceResourcesTag}).
		Doc("List all applications within the specified namespace").
		Param(webservice.QueryParameter(params.ConditionsParam, "query conditions, connect multiple conditions with commas, equal symbol for exact query, wave symbol for fuzzy query e.g. name~a").
			Required(false).
			DataFormat("key=value,key~value").
			DefaultValue("")).
		Param(webservice.PathParameter("namespace", "the name of the project.").Required(true)).
		Param(webservice.QueryParameter(params.PagingParam, "paging query, e.g. limit=100,page=1").
			Required(false).
			DataFormat("limit=%d,page=%d").
			DefaultValue("limit=10,page=1")))

	webservice.Route(webservice.DELETE("/workspaces/{workspace}/namespaces/{namespace}/applications/{application}").
		To(handler.DeleteApplication).
		Doc("Delete the specified application").
		Metadata(restfulspec.KeyOpenAPITags, []string{constants.NamespaceResourcesTag}).
		Returns(http.StatusOK, api.StatusOK, errors.Error{}).
		Param(webservice.PathParameter("namespace", "the name of the project").Required(true)).
		Param(webservice.PathParameter("workspace", "the workspace of the project").Required(true)).
		Param(webservice.PathParameter("application", "the id of the application").Required(true)))

	webservice.Route(webservice.DELETE("/workspaces/{workspace}/clusters/{cluster}/namespaces/{namespace}/applications/{application}").
		To(handler.DeleteApplication).
		Doc("Delete the specified application").
		Metadata(restfulspec.KeyOpenAPITags, []string{constants.NamespaceResourcesTag}).
		Returns(http.StatusOK, api.StatusOK, errors.Error{}).
		Param(webservice.PathParameter("namespace", "the name of the project").Required(true)).
		Param(webservice.PathParameter("cluster", "the name of the cluster.").Required(true)).
		Param(webservice.PathParameter("workspace", "the workspace of the project").Required(true)).
		Param(webservice.PathParameter("application", "the id of the application").Required(true)))

	container.Add(webservice)
	return nil
}
