package v1alpha1

import (
	"github.com/emicklei/go-restful"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"

	"kubesphere.io/kubesphere/pkg/apiserver/runtime"
)

var GroupVersion = schema.GroupVersion{Group: "platformui.kubesphere.io", Version: "v1alpha1"}

func AddToContainer(c *restful.Container, k8sCli kubernetes.Interface) error {
	h := newPlatformUIHandler(k8sCli)

	webservice := runtime.NewWebService(GroupVersion)

	webservice.Route(webservice.POST("/config").
		To(h.createPlatformUI).
		Doc("create customer platform ui config ").
		Reads(PlatformUIConf{}))

	webservice.Route(webservice.PUT("/config").
		To(h.updatePlatformUI).
		Doc("update customer platform ui config ").
		Reads(PlatformUIConf{}))

	webservice.Route(webservice.GET("/config").
		To(h.getPlatformUI).
		Doc("get customer platform ui config"))

	webservice.Route(webservice.DELETE("/config").
		To(h.deletePlatformUI).
		Doc("delete customer platform ui"))

	c.Add(webservice)
	return nil
}
