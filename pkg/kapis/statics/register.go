package statics

import (
	"github.com/emicklei/go-restful"

	"kubesphere.io/kubesphere/pkg/simple/client/s3"
)

func AddToContainer(c *restful.Container, s3Client s3.Interface) error {
	webservice := new(restful.WebService)
	h := newStaticsHandler(s3Client)
	webservice.Route(webservice.POST("/statics/images").
		Doc("upload statics images").
		Consumes("multipart/form-data").
		To(h.uploadStatics).
		Param(webservice.BodyParameter("image", "statics images,support jpg png svg, size in 2M")))
	webservice.Route(webservice.GET("/statics/images/{name}").
		Doc("get statics images").
		To(h.getStaticsImage).
		Param(webservice.PathParameter("name", "statics image name")))

	c.Add(webservice)
	return nil
}
