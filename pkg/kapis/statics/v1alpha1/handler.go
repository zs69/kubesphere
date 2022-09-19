package v1alpha1

import (
	"errors"

	"github.com/emicklei/go-restful"
	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog"

	ksapi "kubesphere.io/kubesphere/pkg/api"
	"kubesphere.io/kubesphere/pkg/simple/client/s3"
)

const (
	StaticsPath            = "/statics/images/"
	S3KeyStaticsPlatformUI = "platform-ui-statics"

	Size2M int64 = 2 * 1024 * 1024
)

var StaticStyles = sets.NewString("images/png", "images/svg+xml", "images/jpeg")

type handler struct {
	s3Cli s3.Interface
}

func newStaticsHandler(s3Cli s3.Interface) *handler {
	return &handler{s3Cli: s3Cli}
}

func (h handler) uploadStatics(req *restful.Request, resp *restful.Response) {
	err := req.Request.ParseMultipartForm(Size2M)
	if err != nil {
		klog.Error(err)
		ksapi.HandleBadRequest(resp, req, err)
		return
	}
	headers, existed := req.Request.MultipartForm.File["image"]
	if !existed || len(headers) == 0 {
		ksapi.HandleBadRequest(resp, req, errors.New("image filed not existed"))
		return
	}
	header := headers[0]
	contentType := header.Header.Get("Content-Type")
	if !StaticStyles.Has(contentType) {
		ksapi.HandleBadRequest(resp, req, errors.New("not supported file style"))
		return
	}
	f, fErr := header.Open()
	defer f.Close()
	if fErr != nil {
		klog.Error(fErr)
		ksapi.HandleBadRequest(resp, req, fErr)
		return
	}
	fName := randStaticsFileName(contentType)
	err = h.s3Cli.Upload(S3KeyStaticsPlatformUI, fName, f, int(header.Size))
	if err != nil {
		klog.Error(err)
		ksapi.HandleBadRequest(resp, req, err)
		return
	}
	result := map[string]string{"image": StaticsPath + fName}
	resp.WriteAsJson(result)
}

func randStaticsFileName(contentType string) string {
	uid := uuid.New().String()
	switch contentType {
	case "images/png":
		return uid + ".png"
	case "images/jpeg":
		return uid + ".jpg"
	case "images/svg+xml":
		return uid + ".svg"
	default:
		return uid
	}
}
