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

package openpitrix

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"kubesphere.io/kubesphere/pkg/utils/clusterclient"

	"kubesphere.io/kubesphere/pkg/apiserver/query"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"kubesphere.io/api/application/v1alpha1"

	"kubesphere.io/kubesphere/pkg/client/clientset/versioned"
	typed_v1alpha1 "kubesphere.io/kubesphere/pkg/client/clientset/versioned/typed/application/v1alpha1"
	"kubesphere.io/kubesphere/pkg/client/informers/externalversions"
	listers_v1alpha1 "kubesphere.io/kubesphere/pkg/client/listers/application/v1alpha1"
	"kubesphere.io/kubesphere/pkg/constants"
	"kubesphere.io/kubesphere/pkg/models"
	"kubesphere.io/kubesphere/pkg/server/params"
	"kubesphere.io/kubesphere/pkg/utils/stringutils"
)

type ManifestInterface interface {
	CreateManifest(workspace, clusterName, namespace string, request CreateManifestRequest) error
	DeleteManifest(workspace, clusterName, namespace, manifestName string) error
	ModifyManifest(request ModifyManifestRequest) error
	ListManifests(workspace, clusterName, namespace string, conditions *params.Conditions, limit, offset int, orderBy string, reverse bool) (*models.PageableResponse, error)
	DescribeManifest(workspace, clusterName, namespace, manifestName string) (*Manifest, error)
}

type manifestOperator struct {
	informers         informers.SharedInformerFactory
	manifestClient    typed_v1alpha1.ManifestInterface
	manifestLister    listers_v1alpha1.ManifestLister
	operatorVerClient typed_v1alpha1.OperatorApplicationVersionInterface
	operatorClient    typed_v1alpha1.OperatorApplicationInterface
	operatorVerLister listers_v1alpha1.OperatorApplicationVersionLister
	operatorLister    listers_v1alpha1.OperatorApplicationLister
	clusterClients    clusterclient.ClusterClients
}

func newManifestOperator(k8sFactory informers.SharedInformerFactory, ksFactory externalversions.SharedInformerFactory, ksClient versioned.Interface) ManifestInterface {
	m := &manifestOperator{
		informers:         k8sFactory,
		manifestClient:    ksClient.ApplicationV1alpha1().Manifests(),
		operatorVerClient: ksClient.ApplicationV1alpha1().OperatorApplicationVersions(),
		operatorClient:    ksClient.ApplicationV1alpha1().OperatorApplications(),
		manifestLister:    ksFactory.Application().V1alpha1().Manifests().Lister(),
		operatorVerLister: ksFactory.Application().V1alpha1().OperatorApplicationVersions().Lister(),
		operatorLister:    ksFactory.Application().V1alpha1().OperatorApplications().Lister(),
		clusterClients:    clusterclient.NewClusterClient(ksFactory.Cluster().V1alpha1().Clusters()),
	}
	return m
}

func (c *manifestOperator) CreateManifest(workspace, clusterName, namespace string, request CreateManifestRequest) error {
	if len(request.Name) > 32 {
		return errors.New("the cluster name cannot exceed 32 characters")
	}

	exists, err := c.manifestExists(workspace, clusterName, namespace, request.Name)

	if err != nil && !apierrors.IsNotFound(err) {
		klog.Errorf("get manifest %s failed, error: %v", clusterName, err)
		return err
	}

	if exists {
		err = fmt.Errorf("manifest %s exists", request.Name)
		klog.Error(err)
		return err
	}

	manifest := &v1alpha1.Manifest{
		ObjectMeta: metav1.ObjectMeta{
			Name: request.Name,
			Annotations: map[string]string{
				constants.CreatorAnnotationKey: request.Username,
			},
			Labels: map[string]string{
				constants.WorkspaceLabelKey: workspace,
				constants.NamespaceLabelKey: namespace,
			},
		},
		Spec: v1alpha1.ManifestSpec{
			Cluster:          clusterName,
			Namespace:        namespace,
			Description:      request.Description,
			AppVersion:       request.AppVersion,
			Version:          request.Version,
			CustomResource:   request.CustomResource,
			RelatedResources: request.RelatedResources,
		},
	}
	if clusterName != "" {
		manifest.Labels[constants.ClusterNameLabelKey] = clusterName
	}
	manifest, err = c.manifestClient.Create(context.TODO(), manifest, metav1.CreateOptions{})

	if err != nil {
		klog.Errorln(err)
		return err
	} else {
		klog.Infof("create manifest %s success in %s", request.Name, clusterName)
	}

	return nil
}

func (c *manifestOperator) DeleteManifest(workspace, clusterName, namespace, manifestName string) error {

	_, err := c.manifestLister.Get(manifestName)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		klog.Errorf("get manifest %s failed, err: %s", manifestName, err)
		return err
	}

	err = c.manifestClient.Delete(context.TODO(), manifestName, metav1.DeleteOptions{})

	if err != nil {
		klog.Errorf("delete manifest %s failed, error: %s", manifestName, err)
		return err
	} else {
		klog.V(2).Infof("delete manifest %s", manifestName)
	}

	return nil
}

func (c *manifestOperator) ModifyManifest(request ModifyManifestRequest) error {

	manifest, err := c.manifestLister.Get(request.Name)
	if err != nil {
		klog.Errorf("get release failed, error: %s", err)
		return err
	}

	manifestCopy := manifest.DeepCopy()
	if request.Description != "" {
		manifestCopy.Spec.Description = stringutils.ShortenString(strings.TrimSpace(request.Description), v1alpha1.MsgLen)
	}

	// update customResource
	if manifest.Spec.Version != request.Version {
		manifestCopy.Spec.Description = request.Description
		manifestCopy.Spec.Version = request.Version
		manifestCopy.Spec.CustomResource = request.CustomResource
	}

	pt := client.MergeFrom(manifest)

	data, err := pt.Data(manifestCopy)
	if err != nil {
		klog.Errorf("create patch failed, error: %s", err)
		return err
	}

	_, err = c.manifestClient.Patch(context.TODO(), request.Name, pt.Type(), data, metav1.PatchOptions{})
	if err != nil {
		klog.Errorln(err)
		return err
	}

	return nil
}

func (c *manifestOperator) ListManifests(workspace, clusterName, namespace string, conditions *params.Conditions, limit, offset int, orderBy string, reverse bool) (*models.PageableResponse, error) {
	ls := map[string]string{}
	if workspace != "" {
		ls[constants.WorkspaceLabelKey] = workspace
	}
	if namespace != "" {
		ls[constants.NamespaceLabelKey] = namespace
	}
	if clusterName != "" {
		ls[constants.ClusterNameLabelKey] = clusterName
	}

	manifests, err := c.manifestLister.List(labels.SelectorFromSet(ls))
	if err != nil && !apierrors.IsNotFound(err) {
		klog.Errorf("list app release failed, error: %v", err)
		return nil, err
	}

	manifests = filterManifests(manifests, conditions)

	if reverse {
		sort.Sort(sort.Reverse(ManifestList(manifests)))
	} else {
		sort.Sort(ManifestList(manifests))
	}

	totalCount := len(manifests)
	start, end := (&query.Pagination{Limit: limit, Offset: offset}).GetValidPagination(totalCount)
	manifests = manifests[start:end]
	items := make([]interface{}, 0, len(manifests))
	for i := range manifests {
		mft := convertManifest(manifests[i])
		items = append(items, mft)
	}

	return &models.PageableResponse{TotalCount: totalCount, Items: items}, nil
}

func (c *manifestOperator) DescribeManifest(workspace, clusterName, namespace, manifestName string) (*Manifest, error) {
	mft, err := c.manifestLister.Get(manifestName)
	if err != nil && !apierrors.IsNotFound(err) {
		klog.Errorf("list manifest failed, error: %v", err)
		return nil, err
	}
	app := &Manifest{}
	if mft != nil {
		app = convertManifest(mft)
	}

	return app, nil
}

func (c *manifestOperator) manifestExists(workspace, clusterName, namespace, name string) (bool, error) {
	set := map[string]string{
		constants.WorkspaceLabelKey: workspace,
		constants.NamespaceLabelKey: namespace,
	}
	if clusterName != "" {
		set[constants.ClusterNameLabelKey] = clusterName
	}

	list, err := c.manifestLister.List(labels.SelectorFromSet(set))
	if err != nil {
		return false, err
	}
	for _, manifest := range list {
		if manifest.Name == name {
			return true, nil
		}
	}
	return false, nil
}

// get operator application version
func (c *manifestOperator) getOperatorAppVersion(versionName string) (ret *v1alpha1.OperatorApplicationVersion, err error) {
	ret, err = c.operatorVerLister.Get(versionName)
	if err != nil && !apierrors.IsNotFound(err) {
		klog.Error(err)
		return nil, err
	}
	return
}
