/*
Copyright 2020 The KubeSphere Authors.

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
	"github.com/emicklei/go-restful"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"

	kubesphereconfig "kubesphere.io/kubesphere/pkg/apiserver/config"
	"kubesphere.io/kubesphere/pkg/apiserver/runtime"
	"kubesphere.io/kubesphere/pkg/simple/client/gpu"
)

const (
	GroupName = "config.kubesphere.io"
)

var GroupVersion = schema.GroupVersion{Group: GroupName, Version: "v1alpha2"}

func AddToContainer(c *restful.Container, config *kubesphereconfig.Config, k8sCLi kubernetes.Interface) error {
	webservice := runtime.NewWebService(GroupVersion)
	h := newPlatformUIHandler(k8sCLi)
	webservice.Route(webservice.GET("/configs/oauth").
		Doc("Information about the authorization server are published.").
		To(func(request *restful.Request, response *restful.Response) {
			response.WriteEntity(config.AuthenticationOptions.OAuthOptions)
		}))

	webservice.Route(webservice.GET("/configs/configz").
		Doc("Information about the server configuration").
		To(func(request *restful.Request, response *restful.Response) {
			response.WriteAsJson(config.ToMap())
		}))

	webservice.Route(webservice.GET("/configs/gpu/kinds").
		Doc("Get all supported GPU kinds.").
		To(func(request *restful.Request, response *restful.Response) {
			var kinds []gpu.GPUKind
			if config.GPUOptions != nil {
				kinds = config.GPUOptions.Kinds
			}
			response.WriteAsJson(kinds)
		}))

	webservice.Route(webservice.POST("/configs/theme").
		To(h.createPlatformUI).
		Doc("create customer platform ui config ").
		Reads(PlatformUIConf{}))

	webservice.Route(webservice.PUT("/configs/theme").
		To(h.updatePlatformUI).
		Doc("update customer platform ui config ").
		Reads(PlatformUIConf{}))

	webservice.Route(webservice.GET("/configs/theme").
		To(h.getPlatformUI).
		Doc("get customer platform ui config"))

	webservice.Route(webservice.DELETE("/configs/theme").
		To(h.deletePlatformUI).
		Doc("delete customer platform ui"))
	c.Add(webservice)
	return nil
}
