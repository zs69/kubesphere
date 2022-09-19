package v1alpha1

import (
	"github.com/emicklei/go-restful"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"kubesphere.io/kubesphere/pkg/apiserver/runtime"
	"kubesphere.io/kubesphere/pkg/simple/client/s3"
)

var GroupVersion = schema.GroupVersion{Group: "statics.kubesphere.io", Version: "v1alpha1"}

func AddToContainer(c *restful.Container, s3Client s3.Interface) error {
	webservice := runtime.NewWebService(GroupVersion)

	h := newStaticsHandler(s3Client)

	webservice.Route(webservice.POST("/statics/images").
		Doc("upload statics images").
		Consumes("multipart/form-data").
		To(h.uploadStatics).
		Param(webservice.BodyParameter("image", "statics images,support jpg png svg, size in 2M")))
	webservice.Route(webservice.GET("/statics/images/{name}").
		Doc("get statics images").
		To(h.getStaticsImage).
		Param(webservice.BodyParameter("image", "statics images,support jpg png svg, size in 2M")))

	c.Add(webservice)
	return nil
}
