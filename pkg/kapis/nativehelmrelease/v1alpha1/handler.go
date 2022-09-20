/*
Copyright 2022 KubeSphere Authors

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
	"bytes"
	"context"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"k8s.io/apimachinery/pkg/watch"

	"helm.sh/helm/v3/pkg/release"

	restful "github.com/emicklei/go-restful"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	v1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
	"kubesphere.io/api/application/v1alpha1"

	"kubesphere.io/kubesphere/pkg/api"
	"kubesphere.io/kubesphere/pkg/apiserver/query"
	"kubesphere.io/kubesphere/pkg/client/clientset/versioned"
	typed_v1alpha1 "kubesphere.io/kubesphere/pkg/client/clientset/versioned/typed/application/v1alpha1"
	"kubesphere.io/kubesphere/pkg/client/informers/externalversions"
	listers_v1alpha1 "kubesphere.io/kubesphere/pkg/client/listers/application/v1alpha1"
	"kubesphere.io/kubesphere/pkg/constants"
	"kubesphere.io/kubesphere/pkg/models"
	"kubesphere.io/kubesphere/pkg/models/helmcmdrelease"
	"kubesphere.io/kubesphere/pkg/models/openpitrix"
	"kubesphere.io/kubesphere/pkg/models/resources/v1alpha3"
	"kubesphere.io/kubesphere/pkg/server/errors"
	"kubesphere.io/kubesphere/pkg/server/params"
	"kubesphere.io/kubesphere/pkg/simple/client/openpitrix/helmwrapper"
	"kubesphere.io/kubesphere/pkg/utils/clusterclient"
	"kubesphere.io/kubesphere/pkg/utils/resourceparse"
)

type clusterClients struct {
	clusterclient.ClusterClients
}

type event struct {
	Type   watch.EventType       `json:"type"`
	Object *v1alpha1.HelmRelease `json:"object"`
}

type watcher interface {
	Stop()
	ResultChan() <-chan event
	AddedEventChan() <-chan event
}

type Watch struct {
	h   *nativeHelmReleaseHandler
	num int
	ns  string
}

func (w *Watch) Stop() {
	w.h.watcherMutex.Lock()
	defer w.h.watcherMutex.Unlock()
	delete(w.h.listWatcher, w.num)
	delete(w.h.addedEventChan, w.num)
}

func (w *Watch) ResultChan() <-chan event {
	w.h.watcherMutex.RLock()
	defer w.h.watcherMutex.RUnlock()
	return w.h.listWatcher[w.num]
}

func (w *Watch) AddedEventChan() <-chan event {
	w.h.watcherMutex.RLock()
	defer w.h.watcherMutex.RUnlock()
	return w.h.addedEventChan[w.num]
}

type nativeHelmReleaseHandler struct {
	secretInformer    v1.SecretInformer
	configMapInformer v1.ConfigMapInformer
	cache             map[string]map[string]*v1alpha1.HelmRelease

	rlsClient typed_v1alpha1.HelmReleaseInterface
	rlsLister listers_v1alpha1.HelmReleaseLister

	listWatcher    map[int]chan event
	addedEventChan map[int]chan event
	eventCh        chan event
	watcherMutex   sync.RWMutex
	watcherCount   int

	clusterClients
	sync.RWMutex
}

type ReleaseNamespacedLister interface {
	List(selector labels.Selector) (ret []*v1alpha1.HelmRelease, err error)
	Get(name string) (*v1alpha1.HelmRelease, error)
}

type namespacedLister struct {
	h         *nativeHelmReleaseHandler
	namespace string
}

func (nl *namespacedLister) List(selector labels.Selector) (ret []*v1alpha1.HelmRelease, err error) {
	if v, ok := nl.h.cache[nl.namespace]; ok {
		if selector.Empty() {
			ret = make([]*v1alpha1.HelmRelease, 0, len(nl.h.cache[nl.namespace]))
			for key := range v {
				ret = append(ret, v[key])
			}
			return
		}
		// TODO, add selector
	}
	return make([]*v1alpha1.HelmRelease, 0), nil
}

func (nl *namespacedLister) Get(name string) (*v1alpha1.HelmRelease, error) {
	if r, ok := nl.h.cache[nl.namespace][name]; ok {
		return r, nil
	} else {
		return nil, apierrors.NewNotFound(schema.GroupResource{
			Group:    GroupName,
			Resource: Resource,
		}, name)
	}
}

func (h *nativeHelmReleaseHandler) Releases(ns string) ReleaseNamespacedLister {
	return &namespacedLister{h: h, namespace: ns}
}

func (h *nativeHelmReleaseHandler) Run() {
	h.secretInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			secret := obj.(*corev1.Secret)
			if rls, err := helmcmdrelease.ToReleaseCR(secret); err != nil {
				if err == helmcmdrelease.ErrNotHelmRelease {
					klog.V(4).Infof("secret %s/%s is not a helm release", secret.Namespace, secret.Name)
					return
				}
				klog.Errorf("failed to convert secret %s/%s to a helm release", secret.Namespace, secret.Name)
			} else if rls.Status.State != string(release.StatusSuperseded) {

				if version, err := helmcmdrelease.ReleaseVersion(secret); err != nil {
					klog.Errorf("get helm release %s/%s version failed", rls.Namespace, rls.Name)
				} else if version >= rls.Spec.Version {
					h.Lock()
					if h.cache[secret.Namespace] == nil {
						h.cache[secret.Namespace] = make(map[string]*v1alpha1.HelmRelease, 4)
					}
					// A new version.
					h.cache[rls.GetRlsNamespace()][rls.Spec.Name] = rls
					h.Unlock()

					h.eventCh <- event{Type: watch.Added, Object: rls}
				}
				klog.V(2).Infof("save helm release %s/%s to cache", rls.Namespace, rls.Name)
			}
		},

		DeleteFunc: func(obj interface{}) {
			secret := obj.(*corev1.Secret)
			if rls, err := helmcmdrelease.ToReleaseCR(secret); err != nil {
				if err == helmcmdrelease.ErrNotHelmRelease {
					klog.V(4).Infof("secret %s/%s is not a helm release", secret.Namespace, secret.Name)
					return
				}
				klog.Errorf("failed to convert secret %s/%s to a helm release", secret.Namespace, secret.Name)
			} else if rls.Status.State != string(release.StatusSuperseded) {
				if version, err := helmcmdrelease.ReleaseVersion(secret); err != nil {
					klog.Errorf("get helm release %s/%s version failed", rls.Namespace, rls.Name)
				} else if version == rls.Spec.Version {
					h.Lock()
					// remove the exact version
					delete(h.cache[rls.GetRlsNamespace()], rls.Spec.Name)
					h.Unlock()
					h.eventCh <- event{Type: watch.Deleted, Object: rls}
				}
				klog.V(2).Infof("delete helm release %s/%s from cache", rls.Namespace, rls.Name)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			secret := newObj.(*corev1.Secret)
			if rls, err := helmcmdrelease.ToReleaseCR(secret); err != nil {
				if err == helmcmdrelease.ErrNotHelmRelease {
					klog.V(4).Infof("secret %s/%s is not a helm release", secret.Namespace, secret.Name)
					return
				}
				klog.Errorf("failed to convert secret %s/%s to a helm release", secret.Namespace, secret.Name)
			} else if rls.Status.State != string(release.StatusSuperseded) {
				if version, err := helmcmdrelease.ReleaseVersion(secret); err != nil {
					klog.Errorf("get helm release %s/%s version failed", rls.Namespace, rls.Name)
				} else if version >= rls.Spec.Version {
					h.Lock()
					h.cache[rls.GetRlsNamespace()][rls.Spec.Name] = rls
					h.Unlock()
					h.eventCh <- event{Type: watch.Modified, Object: rls}
				}
				klog.V(2).Infof("update helm release %s/%s in cache", rls.Namespace, rls.Name)
			}
		},
	})
}

func newHelmShReleaseHandler(ksFactory externalversions.SharedInformerFactory, ksClient versioned.Interface, cc clusterclient.ClusterClients, secretInformer v1.SecretInformer, configMapInformer v1.ConfigMapInformer) *nativeHelmReleaseHandler {
	handler := &nativeHelmReleaseHandler{
		secretInformer:    secretInformer,
		configMapInformer: configMapInformer,
		cache:             make(map[string]map[string]*v1alpha1.HelmRelease),
		rlsClient:         ksClient.ApplicationV1alpha1().HelmReleases(),
		rlsLister:         ksFactory.Application().V1alpha1().HelmReleases().Lister(),
		clusterClients:    clusterClients{cc},
		eventCh:           make(chan event, 30),
		listWatcher:       make(map[int]chan event, 20),
		addedEventChan:    make(map[int]chan event, 20),
	}

	go handler.Run()
	go handler.EventLoop()
	return handler
}

// EventLoop sends all the new events to
// the registered channel.
func (h *nativeHelmReleaseHandler) EventLoop() {
	for e := range h.eventCh {
		h.watcherMutex.RLock()
		for id, ch := range h.listWatcher {
			select {
			case ch <- e:
			default:
				klog.Warningf("channel %d has been blocked, release: %s/%s", id, e.Object.GetRlsNamespace(), e.Object.GetTrueName())
			}
		}
		h.watcherMutex.RUnlock()
	}
}

func (h *nativeHelmReleaseHandler) NewWatcher(ns string) watcher {
	h.RLock()
	all, _ := h.Releases(ns).List(labels.Everything())
	w := &Watch{
		h:   h,
		num: h.watcherCount,
		ns:  ns,
	}

	h.watcherMutex.Lock()
	c := make(chan event, 4)
	h.listWatcher[h.watcherCount] = c

	addedEvent := make(chan event, 2)
	h.addedEventChan[h.watcherCount] = addedEvent
	h.watcherMutex.Unlock()

	h.watcherCount++
	h.RUnlock()

	// Add existing release to the channel.
	go func() {
		for ind := range all {
			addedEvent <- event{Type: watch.Added, Object: all[ind]}
		}
		close(addedEvent)
	}()

	return w
}

func (h *nativeHelmReleaseHandler) ListReleases(req *restful.Request, resp *restful.Response) {
	namespace := req.PathParameter("namespace")
	qry := query.ParseQueryParameter(req)
	for f := range qry.Filters {
		if string(f) == "watch" {
			delete(qry.Filters, f)
			break
		}
	}

	w, _ := strconv.ParseBool(req.QueryParameter("watch"))

	if w {
		watcher := h.NewWatcher(namespace)
		defer watcher.Stop()
		upgrader := websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			// Allow connections from any Origin
			CheckOrigin: func(r *http.Request) bool { return true },
		}

		conn, err := upgrader.Upgrade(resp, req.Request, nil)
		if err != nil {
			apierrors.NewInternalError(err)
			return
		}

		done := req.Request.Context().Done()
		var resultChan <-chan event
		addedEventChan := watcher.AddedEventChan()
		for {
			select {
			case <-done:
				conn.Close()
				return
			case v, ok := <-addedEventChan:
				if !ok {
					// all initial events sent.
					resultChan = watcher.ResultChan()
					addedEventChan = nil
					continue
				}
				ns := v.Object.GetRlsNamespace()
				if ns == namespace && match(v.Object, qry, h.filter) {
					conn.WriteJSON(v)
				}
			case v, ok := <-resultChan:
				if !ok {
					conn.Close()
					return
				}
				ns := v.Object.GetRlsNamespace()
				if ns == namespace && match(v.Object, qry, h.filter) {
					conn.WriteJSON(v)
				}
			}
		}
	}

	ret, err := h.Search(namespace, qry)
	if err != nil {
		klog.Errorf("search release cr failed, error: %s", err)
		apierrors.NewInternalError(err)
		return
	}

	resp.WriteEntity(ret)
	return
}

func (h *nativeHelmReleaseHandler) Search(namespace string, query *query.Query) (*api.ListResult, error) {
	h.RLock()
	allRls, err := h.Releases(namespace).List(labels.Everything())
	h.RUnlock()

	if err != nil {
		return nil, err
	}
	var result []runtime.Object
	for i := range allRls {
		result = append(result, allRls[i])
	}
	return v1alpha3.DefaultList(result, query, h.compare, h.filter), nil
}

func match(obj runtime.Object, q *query.Query, filter v1alpha3.FilterFunc) bool {
	selected := true
	if q == nil {
		return selected
	}
	for field, value := range q.Filters {
		if !filter(obj, query.Filter{Field: field, Value: value}) {
			selected = false
			break
		}
	}

	return selected
}

func (h *nativeHelmReleaseHandler) Get(req *restful.Request, resp *restful.Response) {
	ns := req.PathParameter("namespace")
	name := req.PathParameter("name")

	h.RLock()
	defer h.RUnlock()
	if rls, err := h.Releases(ns).Get(name); err == nil {
		resp.WriteEntity(rls)
		return
	} else {
		if apierrors.IsNotFound(err) {
			api.HandleNotFound(resp, nil, err)
		} else {
			api.HandleInternalError(resp, nil, err)
		}
		return
	}
}

func (h *nativeHelmReleaseHandler) Delete(req *restful.Request, resp *restful.Response) {
	ns := req.PathParameter("namespace")
	name := req.PathParameter("name")

	h.RLock()
	_, err := h.Releases(ns).Get(name)
	h.RUnlock()

	if err == nil {
		wrapper := helmwrapper.NewHelmWrapper("", ns, name)
		if err = wrapper.Uninstall(); err != nil {
			api.HandleInternalError(resp, nil, err)
			return
		}
	} else {
		if apierrors.IsNotFound(err) {
			return
		} else {
			api.HandleInternalError(resp, nil, err)
		}
		return
	}
}

func (h *nativeHelmReleaseHandler) compare(left runtime.Object, right runtime.Object, field query.Field) bool {
	leftRls, ok := left.(*v1alpha1.HelmRelease)
	if !ok {
		return false
	}

	rightRls, ok := right.(*v1alpha1.HelmRelease)
	if !ok {
		return false
	}
	switch field {
	case query.FieldName:
		return strings.Compare(leftRls.Spec.Name, rightRls.Spec.Name) > 0
	default:
		return v1alpha3.DefaultObjectMetaCompare(leftRls.ObjectMeta, rightRls.ObjectMeta, field)
	}
}

func (h *nativeHelmReleaseHandler) filter(object runtime.Object, filter query.Filter) bool {
	rls, ok := object.(*v1alpha1.HelmRelease)
	if !ok {
		return false
	}

	switch filter.Field {
	case query.FieldName:
		return strings.Contains(rls.Spec.Name, string(filter.Value))
	case query.FieldStatus:
		return strings.Contains(rls.Status.State, string(filter.Value))
	default:
		return v1alpha3.DefaultObjectMetaFilter(rls.ObjectMeta, filter)
	}
}

// DescribeApplication returns the same info with interface in pkg/kapis/openpitrix/v1
func (h *nativeHelmReleaseHandler) DescribeApplication(req *restful.Request, resp *restful.Response) {
	namespace := req.PathParameter("namespace")
	name := req.PathParameter("application")

	app, err := h.describeApplication(namespace, name)

	if err != nil {
		if apierrors.IsNotFound(err) {
			api.HandleNotFound(resp, nil, err)
			return
		}
		klog.Errorln(err)
		api.HandleInternalError(resp, nil, err)
		return
	}

	resp.WriteEntity(app)
	return
}

func (h *nativeHelmReleaseHandler) describeApplication(namespace, name string) (*openpitrix.Application, error) {
	h.RLock()
	rls, err := h.Releases(namespace).Get(name)
	h.RUnlock()
	if err != nil {
		klog.Errorf("get helm release %s/%s failed, error: %v", namespace, name, err)
		return nil, err
	}

	app := &openpitrix.Application{}

	if rls != nil {
		hw := helmwrapper.NewHelmWrapper("", namespace, rls.Spec.Name)
		manifest, err := hw.Manifest()
		if err != nil {
			klog.Errorf("get manifest of helm release %s/%s failed, error: %s", namespace, name, err)
		}
		infos, err := resourceparse.Parse(bytes.NewBufferString(manifest), namespace, rls.Spec.Name, true)
		if err != nil {
			klog.Errorf("parse resource failed, error: %s", err)
		}
		app = openpitrix.ConvertApplication(rls, infos)
	}

	return app, nil
}

// ListApplications returns info compatible with interface in pkg/kapis/openpitrix/v1.
func (h *nativeHelmReleaseHandler) ListApplications(req *restful.Request, resp *restful.Response) {
	limit, offset := params.ParsePaging(req)
	namespace := req.PathParameter("namespace")
	orderBy := params.GetStringValueWithDefault(req, params.OrderByParam, openpitrix.StatusTime)
	conditions, err := params.ParseConditions(req)
	if err != nil {
		klog.V(4).Infoln(err)
		api.HandleBadRequest(resp, nil, err)
		return
	}

	reverse := false
	if conditions.Match[openpitrix.Ascending] == "true" {
		reverse = true
	}

	result, err := h.listApplications(namespace, conditions, limit, offset, orderBy, reverse)

	if err != nil {
		klog.Errorln(err)
		api.HandleInternalError(resp, nil, err)
		return
	}

	resp.WriteEntity(result)
}

func (h *nativeHelmReleaseHandler) listApplications(namespace string, conditions *params.Conditions, limit, offset int, orderBy string, reverse bool) (*models.PageableResponse, error) {

	var err error
	h.RLock()
	releases, err := h.Releases(namespace).List(labels.Everything())
	h.RUnlock()

	if err != nil && !apierrors.IsNotFound(err) {
		klog.Errorf("list app release failed, error: %v", err)
		return nil, err
	}

	releases = openpitrix.FilterReleases(releases, conditions)

	if reverse {
		sort.Sort(sort.Reverse(openpitrix.HelmReleaseList(releases)))
	} else {
		sort.Sort(openpitrix.HelmReleaseList(releases))
	}

	totalCount := len(releases)
	start, end := (&query.Pagination{Limit: limit, Offset: offset}).GetValidPagination(totalCount)
	releases = releases[start:end]
	items := make([]interface{}, 0, len(releases))
	for i := range releases {
		app := openpitrix.ConvertApplication(releases[i], nil)
		items = append(items, app)
	}

	return &models.PageableResponse{TotalCount: totalCount, Items: items}, nil
}

// DeleteApplication delete helm release from host cluster.
func (h *nativeHelmReleaseHandler) DeleteApplication(req *restful.Request, resp *restful.Response) {
	clusterName := req.PathParameter("cluster")
	workspace := req.PathParameter("workspace")
	applicationId := req.PathParameter("application")
	namespace := req.PathParameter("namespace")

	err := h.deleteApplication(clusterName, workspace, namespace, applicationId)

	if err != nil {
		resp.WriteEntity(errors.Wrap(err))
		return
	}

	resp.WriteEntity(errors.None)
}

func (h *nativeHelmReleaseHandler) deleteApplication(cluster, workspace, namespace, name string) error {
	var err error
	cluster, err = h.rewriteClusterName(cluster)
	if err != nil {
		return err
	}
	ls := map[string]string{}
	if workspace != "" {
		ls[constants.WorkspaceLabelKey] = workspace
	}
	if namespace != "" {
		ls[constants.NamespaceLabelKey] = namespace
	}
	if cluster != "" {
		ls[constants.ClusterNameLabelKey] = cluster
	}

	releases, err := h.rlsLister.List(labels.SelectorFromSet(ls))
	var appId string
	for _, rls := range releases {
		if rls.Spec.Name == name {
			appId = rls.Name
			break
		}
	}

	if appId != "" {
		err = h.rlsClient.Delete(context.TODO(), appId, metav1.DeleteOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return nil
			}
			klog.Errorf("delete release %s/%s failed, error: %s", namespace, name, err)
			return err
		} else {
			klog.V(2).Infof("delete release %s/%s", namespace, name)
		}
	} else {
		// delete a helm cmd release to the specific cluster
		var clusterConfig string
		if cluster != "" {
			clusterConfig, err = h.clusterClients.GetClusterKubeconfig(cluster)
			if err != nil {
				klog.Errorf("get cluster %s config failed, error: %s", cluster, err)
				return err
			}
		}
		wrapper := helmwrapper.NewHelmWrapper(clusterConfig, namespace, name)
		if err = wrapper.Uninstall(); err != nil {
			klog.Errorf("uninstall helm release %s/%s failed, error: %s", namespace, name, err)
		} else {
			klog.Infof("uninstall helm release %s/%s", namespace, name)
		}
	}
	return err
}

func (c *clusterClients) rewriteClusterName(clusterName string) (string, error) {
	if c.ClusterClients == nil || clusterName == "" {
		return clusterName, nil
	} else {
		cluster, err := c.Get(clusterName)
		if err != nil {
			klog.Errorf("get cluster %s failed, error: %v", clusterName, err)
			return clusterName, err
		}

		if c.IsHostCluster(cluster) {
			clusterName = ""
		}
		return clusterName, nil
	}
}
