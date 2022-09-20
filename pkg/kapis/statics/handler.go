package statics

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/emicklei/go-restful"
	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog"

	ksapi "kubesphere.io/kubesphere/pkg/api"
	"kubesphere.io/kubesphere/pkg/simple/client/s3"
)

const (
	StaticsPath = "/statics/images/"

	ImageStylePNG = "images/png"
	ImageStyleJPG = "images/jpeg"
	ImageStyleSVG = "images/svg+xml"

	Size2M int64 = 2 * 1024 * 1024
)

var StaticStyles = sets.NewString(ImageStylePNG, ImageStyleJPG, ImageStyleSVG)

type handler struct {
	s3Client s3.Interface
}

func newStaticsHandler(s3Client s3.Interface) *handler {
	return &handler{s3Client: s3Client}
}

func (h handler) uploadStatics(req *restful.Request, resp *restful.Response) {
	err := req.Request.ParseMultipartForm(Size2M)
	if err != nil {
		klog.Error(err)
		ksapi.HandleBadRequest(resp, req, err)
		return
	}
	fileHeaders, existed := req.Request.MultipartForm.File["image"]
	if !existed || len(fileHeaders) == 0 {
		ksapi.HandleBadRequest(resp, req, errors.New("image filed not existed"))
		return
	}
	fileHeader := fileHeaders[0]
	contentType := fileHeader.Header.Get("Content-Type")
	if !StaticStyles.Has(contentType) {
		ksapi.HandleBadRequest(resp, req, errors.New("not supported fileHeader style"))
		return
	}
	file, fileErr := fileHeader.Open()
	defer file.Close()
	if fileErr != nil {
		klog.Error(fileErr)
		ksapi.HandleBadRequest(resp, req, fileErr)
		return
	}
	fileName, fileType := randStaticsFileName(contentType)
	err = h.s3Client.Upload(fileName, fileName+fileType, file, int(fileHeader.Size))
	if err != nil {
		klog.Error(err)
		ksapi.HandleBadRequest(resp, req, err)
		return
	}
	result := map[string]string{"image": StaticsPath + fileName}
	resp.WriteAsJson(result)
}

func (h handler) getStaticsImage(req *restful.Request, resp *restful.Response) {
	fileName := req.PathParameter("name")
	nameAndSuffix := strings.Split(fileName, ".")

	if len(nameAndSuffix) != 2 {
		ksapi.HandleBadRequest(resp, req, errors.New("invalid filename"))
		return
	}
	url, err := h.s3Client.GetDownloadURL(nameAndSuffix[0], fileName)
	if err != nil {
		klog.Error(err)
		ksapi.HandleBadRequest(resp, req, err)
	}
	img, err := http.Get(url)
	if err != nil {
		klog.Error(err)
		ksapi.HandleInternalError(resp, req, err)
		return
	}
	defer img.Body.Close()
	io.Copy(resp.ResponseWriter, img.Body)
}

func randStaticsFileName(contentType string) (filename, style string) {
	uid := uuid.New().String()
	switch contentType {
	case ImageStylePNG:
		return uid, ".png"
	case ImageStyleJPG:
		return uid, ".jpg"
	case ImageStyleSVG:
		return uid, ".svg"
	default:
		return uid, ""
	}
}
