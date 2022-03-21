package multuscni

import (
	v1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	informers "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/client/informers/externalversions"
	"k8s.io/apimachinery/pkg/runtime"

	"kubesphere.io/kubesphere/pkg/api"
	"kubesphere.io/kubesphere/pkg/apiserver/query"
	"kubesphere.io/kubesphere/pkg/models/resources/v1alpha3"
)

type multuscniGetter struct {
	informers informers.SharedInformerFactory
}

func New(informers informers.SharedInformerFactory) v1alpha3.Interface {
	return &multuscniGetter{
		informers: informers,
	}
}

func (m multuscniGetter) Get(namespace, name string) (runtime.Object, error) {
	return m.informers.K8sCniCncfIo().V1().NetworkAttachmentDefinitions().Lister().NetworkAttachmentDefinitions(namespace).Get(name)
}

func (m multuscniGetter) List(namespace string, query *query.Query) (*api.ListResult, error) {
	multuscnis, err := m.informers.K8sCniCncfIo().V1().NetworkAttachmentDefinitions().Lister().NetworkAttachmentDefinitions(namespace).List(query.Selector()) //NetworkAttachmentDefinitions(namespace).
	if err != nil {
		return nil, err
	}

	var result []runtime.Object
	for _, cni := range multuscnis {
		result = append(result, cni)
	}

	return v1alpha3.DefaultList(result, query, m.compare, m.filter), nil
}

func (m multuscniGetter) compare(left runtime.Object, right runtime.Object, field query.Field) bool {
	leftCNI, ok := left.(*v1.NetworkAttachmentDefinition)
	if !ok {
		return false
	}

	rightCNI, ok := right.(*v1.NetworkAttachmentDefinition)
	if !ok {
		return false
	}

	return v1alpha3.DefaultObjectMetaCompare(leftCNI.ObjectMeta, rightCNI.ObjectMeta, field)
}

func (m multuscniGetter) filter(object runtime.Object, filter query.Filter) bool {
	cni, ok := object.(*v1.NetworkAttachmentDefinition)
	if !ok {
		return false
	}

	return v1alpha3.DefaultObjectMetaFilter(cni.ObjectMeta, filter)
}
