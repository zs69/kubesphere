package v1alpha2

import (
	"encoding/json"
	"errors"

	"github.com/emicklei/go-restful"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"

	ksapi "kubesphere.io/kubesphere/pkg/api"
	kubesphereconfig "kubesphere.io/kubesphere/pkg/apiserver/config"
	"kubesphere.io/kubesphere/pkg/simple/client/s3"
)

const (
	NamespaceKubeSphere       = "kubesphere-system"
	ConfigMapKubeSphereConfig = "kubesphere-config"

	S3KeyLogo       = "sys-config-logo"
	S3KeyFavicon    = "sys-config-favicon"
	S3KeyBackground = "sys-config-background"

	S3NameLogo       = "sys-logo"
	S3NameFavicon    = "sys-favicon"
	S3NameBackground = "sys-background"

	Size2M int64 = 2 * 1024 * 1024
)

var StaticStyles = sets.NewString("images/png", "images/svg+xml", "text/html", "images/jpeg")

type handler struct {
	k8sCli kubernetes.Interface
	s3Cli  s3.Interface
	config *kubesphereconfig.Config
}

func newConfigHandler(k8sCli kubernetes.Interface, s3Cli s3.Interface, config *kubesphereconfig.Config) *handler {
	return &handler{k8sCli: k8sCli, s3Cli: s3Cli, config: config}
}

func (h handler) uploadThemeStatics(req *restful.Request, resp *restful.Response) {
	err := req.Request.ParseMultipartForm(Size2M)
	if err != nil {
		klog.V(0).Info(err)
		ksapi.HandleBadRequest(resp, req, err)
		return
	}
	headers, existed := req.Request.MultipartForm.File["data"]
	if !existed || len(headers) == 0 {
		klog.V(0).Info(existed, headers)
		ksapi.HandleBadRequest(resp, req, errors.New("statics data not existed"))
		return
	}
	header := headers[0]
	contentType := header.Header.Get("Content-Type")
	if !StaticStyles.Has(contentType) {
		ksapi.HandleBadRequest(resp, req, errors.New("not supported file style"))
		return
	}
	file, err := header.Open()
	defer file.Close()
	if err != nil {
		klog.V(0).Info(err)
		ksapi.HandleBadRequest(resp, req, err)
		return
	}

	static := req.PathParameter("static")
	key, name := "", ""
	switch static {
	case "logo":
		key, name = S3KeyLogo, S3NameLogo
	case "favicon":
		key, name = S3KeyFavicon, S3NameFavicon
	case "background":
		key, name = S3KeyBackground, S3NameBackground
	default:
		ksapi.HandleBadRequest(resp, req, errors.New("not supported static"))
		return
	}
	err = h.s3Cli.Upload(key, name, file, int(header.Size))
	if err != nil {
		klog.V(0).Info(err)
		ksapi.HandleBadRequest(resp, req, err)
		return
	}
	url, errUrl := h.s3Cli.GetDownloadURL(S3KeyLogo, S3NameLogo)
	if errUrl != nil {
		klog.Error(errUrl)
		ksapi.HandleBadRequest(resp, req, err)
		return
	}
	resp.WriteAsJson(url)
}

func (h handler) updateTheme(req *restful.Request, resp *restful.Response) {
	param := kubesphereconfig.ThemeConfig{}
	err := req.ReadEntity(&param)
	if err != nil {
		klog.Error(err)
		ksapi.HandleBadRequest(resp, req, err)
		return
	}
	if !param.Valid() {
		ksapi.HandleBadRequest(resp, req, errors.New("invalid value"))
		return
	}
	config := kubesphereconfig.Config{}
	cm, err := h.k8sCli.CoreV1().ConfigMaps(NamespaceKubeSphere).Get(req.Request.Context(), ConfigMapKubeSphereConfig, metav1.GetOptions{})
	if err != nil {
		klog.Error(err)
		ksapi.HandleBadRequest(resp, req, err)
		return
	}
	data := cm.Data["kubesphere.yaml"]
	err = json.Unmarshal([]byte(data), &config)
	if err != nil {
		klog.Error(err)
		ksapi.HandleBadRequest(resp, req, err)
		return
	}
	config.ThemeConfig = &param
	dataByt, err := json.Marshal(config)
	if err != nil {
		klog.Error(err)
		ksapi.HandleBadRequest(resp, req, err)
		return
	}
	cm.Data["kubesphere.yaml"] = string(dataByt)
	_, err = h.k8sCli.CoreV1().ConfigMaps(NamespaceKubeSphere).Update(req.Request.Context(), cm, metav1.UpdateOptions{})
	if err != nil {
		klog.Error(err)
		ksapi.HandleBadRequest(resp, req, err)
		return
	}
	resp.WriteAsJson(param)
}

func (h handler) getTheme(req *restful.Request, resp *restful.Response) {
	var logo, favicon, bg string
	_, err := h.s3Cli.Read(S3KeyLogo)
	if err == nil {
		logo, _ = h.s3Cli.GetDownloadURL(S3KeyLogo, S3NameLogo)
	}
	_, err = h.s3Cli.Read(S3KeyFavicon)
	if err == nil {
		favicon, _ = h.s3Cli.GetDownloadURL(S3KeyFavicon, S3NameFavicon)
	}
	_, err = h.s3Cli.Read(S3KeyBackground)
	if err == nil {
		bg, _ = h.s3Cli.GetDownloadURL(S3KeyBackground, S3NameBackground)
	}
	themeConfig := struct {
		Systitle       string `json:"systitle"`
		Sysdescription string `json:"sysdescription"`
		Logo           string `json:"logo"`
		Favicon        string `json:"favicon"`
		Background     string `json:"background"`
	}{h.config.ThemeConfig.SysTitle, h.config.ThemeConfig.SysDescription, logo, favicon, bg}
	resp.WriteAsJson(themeConfig)
}
