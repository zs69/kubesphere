package v2beta2

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/emicklei/go-restful"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	"kubesphere.io/api/notification/v2beta2"

	"kubesphere.io/kubesphere/pkg/api"
	notificationv2beta2 "kubesphere.io/kubesphere/pkg/api/notification/v2beta2"
	"kubesphere.io/kubesphere/pkg/apiserver/query"
	kubesphere "kubesphere.io/kubesphere/pkg/client/clientset/versioned"
	"kubesphere.io/kubesphere/pkg/informers"
	models "kubesphere.io/kubesphere/pkg/models/notification"
	servererr "kubesphere.io/kubesphere/pkg/server/errors"
	notificationclient "kubesphere.io/kubesphere/pkg/simple/client/notification"
)

const (
	VerificationAPIPath = "/api/v2/verify"
)

type Result struct {
	Code    int    `json:"Status"`
	Message string `json:"Message"`
}
type notification struct {
	Config   v2beta2.Config   `json:"config"`
	Receiver v2beta2.Receiver `json:"receiver"`
}

type handler struct {
	operator models.Operator
	option   *notificationclient.Options
}

func newNotificationHandler(
	informers informers.InformerFactory,
	k8sClient kubernetes.Interface,
	ksClient kubesphere.Interface,
	notificationClient notificationclient.Client,
	option *notificationclient.Options) *handler {

	return &handler{
		operator: models.NewOperator(informers, k8sClient, ksClient, notificationClient),
		option:   option,
	}
}

func (h *handler) ListResource(req *restful.Request, resp *restful.Response) {

	user := req.PathParameter("user")
	resource := req.PathParameter("resources")
	subresource := req.QueryParameter("type")
	q := query.ParseQueryParameter(req)

	if !h.operator.IsKnownResource(resource, subresource) {
		api.HandleBadRequest(resp, req, servererr.New("unknown resource type %s/%s", resource, subresource))
		return
	}

	objs, err := h.operator.List(user, resource, subresource, q)
	handleResponse(req, resp, objs, err)
}

func (h *handler) GetResource(req *restful.Request, resp *restful.Response) {

	user := req.PathParameter("user")
	resource := req.PathParameter("resources")
	name := req.PathParameter("name")
	subresource := req.QueryParameter("type")

	if !h.operator.IsKnownResource(resource, subresource) {
		api.HandleBadRequest(resp, req, servererr.New("unknown resource type %s/%s", resource, subresource))
		return
	}

	obj, err := h.operator.Get(user, resource, name, subresource)
	handleResponse(req, resp, obj, err)
}

func (h *handler) CreateResource(req *restful.Request, resp *restful.Response) {

	user := req.PathParameter("user")
	resource := req.PathParameter("resources")

	if !h.operator.IsKnownResource(resource, "") {
		api.HandleBadRequest(resp, req, servererr.New("unknown resource type %s", resource))
		return
	}

	obj := h.operator.GetObject(resource)
	if err := req.ReadEntity(obj); err != nil {
		api.HandleBadRequest(resp, req, err)
		return
	}

	created, err := h.operator.Create(user, resource, obj)
	handleResponse(req, resp, created, err)
}

func (h *handler) UpdateResource(req *restful.Request, resp *restful.Response) {

	user := req.PathParameter("user")
	resource := req.PathParameter("resources")
	name := req.PathParameter("name")

	if !h.operator.IsKnownResource(resource, "") {
		api.HandleBadRequest(resp, req, servererr.New("unknown resource type %s", resource))
		return
	}

	obj := h.operator.GetObject(resource)
	if err := req.ReadEntity(obj); err != nil {
		api.HandleBadRequest(resp, req, err)
		return
	}

	updated, err := h.operator.Update(user, resource, name, obj)
	handleResponse(req, resp, updated, err)
}

func (h *handler) DeleteResource(req *restful.Request, resp *restful.Response) {

	user := req.PathParameter("user")
	resource := req.PathParameter("resources")
	name := req.PathParameter("name")

	if !h.operator.IsKnownResource(resource, "") {
		api.HandleBadRequest(resp, req, servererr.New("unknown resource type %s", resource))
		return
	}

	handleResponse(req, resp, servererr.None, h.operator.Delete(user, resource, name))
}

func (h *handler) SearchNotification(req *restful.Request, resp *restful.Response) {

	queryParam, err := notificationv2beta2.ParseQueryParameter(req)
	if err != nil {
		klog.Errorln(err)
		api.HandleInternalError(resp, req, err)
		return
	}

	result, err := h.operator.SearchNotification(queryParam)
	if err != nil {
		klog.Errorln(err)
		api.HandleInternalError(resp, req, err)
		return
	}

	_ = resp.WriteEntity(result)
}

func (h *handler) ExportNotification(req *restful.Request, resp *restful.Response) {

	queryParam, err := notificationv2beta2.ParseQueryParameter(req)
	if err != nil {
		klog.Errorln(err)
		api.HandleInternalError(resp, req, err)
		return
	}

	if err := h.operator.ExportNotification(queryParam, resp); err != nil {
		klog.Errorln(err)
		api.HandleInternalError(resp, req, err)
		return
	}
}

func (h handler) Verify(request *restful.Request, response *restful.Response) {
	opt := h.option
	if opt == nil || len(opt.Endpoint) == 0 {
		_ = response.WriteAsJson(Result{
			http.StatusBadRequest,
			"Cannot find Notification Manager endpoint",
		})
	}
	host := opt.Endpoint
	notification := notification{}
	reqBody, err := ioutil.ReadAll(request.Request.Body)
	if err != nil {
		_ = response.WriteHeaderAndEntity(http.StatusInternalServerError, err)
		return
	}

	err = json.Unmarshal(reqBody, &notification)
	if err != nil {
		_ = response.WriteHeaderAndEntity(http.StatusInternalServerError, err)
		return
	}

	receiver := notification.Receiver
	user := request.PathParameter("user")

	if receiver.Labels["type"] == "tenant" {
		if user != receiver.Labels["user"] {
			_ = response.WriteAsJson(Result{
				http.StatusForbidden,
				"Permission denied",
			})
			return
		}
	}
	if receiver.Labels["type"] == "global" {
		if user != "" {
			_ = response.WriteAsJson(Result{
				http.StatusForbidden,
				"Permission denied",
			})
			return
		}
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s%s", host, VerificationAPIPath), bytes.NewReader(reqBody))
	if err != nil {
		_ = response.WriteHeaderAndEntity(http.StatusInternalServerError, err)
		return
	}
	req.Header = request.Request.Header

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		_ = response.WriteHeaderAndEntity(http.StatusInternalServerError, err)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		// return 500
		_ = response.WriteHeaderAndEntity(http.StatusInternalServerError, err)
		return
	}

	var result Result
	err = json.Unmarshal(body, &result)
	if err != nil {
		_ = response.WriteHeaderAndEntity(http.StatusInternalServerError, err)
		return
	}

	_ = response.WriteAsJson(result)
}

func handleResponse(req *restful.Request, resp *restful.Response, obj interface{}, err error) {

	if err != nil {
		klog.Error(err)
		if errors.IsNotFound(err) {
			api.HandleNotFound(resp, req, err)
			return
		} else if errors.IsConflict(err) {
			api.HandleConflict(resp, req, err)
			return
		}
		api.HandleBadRequest(resp, req, err)
		return
	}

	_ = resp.WriteEntity(obj)
}
