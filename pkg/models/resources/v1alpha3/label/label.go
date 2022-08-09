/*
Copyright 2022 The KubeSphere Authors.

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

package label

import (
	"k8s.io/apimachinery/pkg/runtime"
	clusterv1alpha1 "kubesphere.io/api/cluster/v1alpha1"

	"kubesphere.io/kubesphere/pkg/api"
	"kubesphere.io/kubesphere/pkg/apiserver/query"
	"kubesphere.io/kubesphere/pkg/client/informers/externalversions"
	"kubesphere.io/kubesphere/pkg/models/resources/v1alpha3"
)

type labelsGetter struct {
	informers externalversions.SharedInformerFactory
}

func New(informers externalversions.SharedInformerFactory) v1alpha3.Interface {
	return &labelsGetter{informers: informers}
}

func (n labelsGetter) Get(_, name string) (runtime.Object, error) {
	return n.informers.Cluster().V1alpha1().Labels().Lister().Get(name)
}

func (n labelsGetter) List(_ string, query *query.Query) (*api.ListResult, error) {
	labels, err := n.informers.Cluster().V1alpha1().Labels().Lister().List(query.Selector())
	if err != nil {
		return nil, err
	}

	var result []runtime.Object
	for _, item := range labels {
		result = append(result, item)
	}

	return v1alpha3.DefaultList(result, query, n.compare, n.filter), nil
}

func (n labelsGetter) filter(item runtime.Object, filter query.Filter) bool {
	label, ok := item.(*clusterv1alpha1.Label)
	if !ok {
		return false
	}

	return v1alpha3.DefaultObjectMetaFilter(label.ObjectMeta, filter)
}

func (n labelsGetter) compare(left runtime.Object, right runtime.Object, field query.Field) bool {
	leftLabel, ok := left.(*clusterv1alpha1.Label)
	if !ok {
		return false
	}

	rightLabel, ok := right.(*clusterv1alpha1.Label)
	if !ok {
		return true
	}
	return v1alpha3.DefaultObjectMetaCompare(leftLabel.ObjectMeta, rightLabel.ObjectMeta, field)
}
