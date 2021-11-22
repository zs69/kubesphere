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

package v1alpha1

import (
	"net/http"

	"kubesphere.io/kubesphere/pkg/informers"
	licenseclient "kubesphere.io/kubesphere/pkg/simple/client/license/client"
	"kubesphere.io/kubesphere/pkg/simple/client/multicluster"

	clientset "k8s.io/client-go/kubernetes"

	restful "github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"kubesphere.io/kubesphere/pkg/api"
	"kubesphere.io/kubesphere/pkg/apiserver/runtime"
)

const (
	GroupName  = "license.kubesphere.io"
	LicenseTag = "license resource"
)

var GroupVersion = schema.GroupVersion{Group: GroupName, Version: "v1alpha1"}

func AddToContainer(c *restful.Container, client clientset.Interface, informerFactory informers.InformerFactory, opts *multicluster.Options) error {
	webservice := runtime.NewWebService(GroupVersion)

	handler := newLicenseHandler(client, informerFactory, opts)

	// In fact, the name parameter is not used now,
	// it is used to support multiple licenses in the future
	webservice.Route(webservice.GET("/licenses/{name}").
		To(handler.GetLicense).
		Doc("Get the license").
		Metadata(restfulspec.KeyOpenAPITags, []string{LicenseTag}).
		Returns(http.StatusOK, api.StatusOK, licenseclient.License{}).
		Returns(http.StatusOK, api.StatusOK, licenseclient.License{}))

	webservice.Route(webservice.POST("/licenses/").
		To(handler.UpdateLicense).
		Reads(licenseclient.License{}).
		Doc("Create the license").
		Returns(http.StatusOK, api.StatusOK, licenseclient.License{}).
		Metadata(restfulspec.KeyOpenAPITags, []string{LicenseTag}))

	webservice.Route(webservice.PUT("/licenses/{name}").
		To(handler.UpdateLicense).
		Reads(licenseclient.License{}).
		Doc("Update the license").
		Returns(http.StatusOK, api.StatusOK, licenseclient.License{}).
		Metadata(restfulspec.KeyOpenAPITags, []string{LicenseTag}))

	webservice.Route(webservice.DELETE("/licenses/{name}").
		To(handler.DeleteLicense).
		Doc("Delete the license").
		Metadata(restfulspec.KeyOpenAPITags, []string{LicenseTag}))

	c.Add(webservice)

	return nil
}
