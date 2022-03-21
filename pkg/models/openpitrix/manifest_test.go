package openpitrix

import (
	"context"
	"encoding/json"
	"testing"

	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakek8s "k8s.io/client-go/kubernetes/fake"
	"k8s.io/klog"

	"kubesphere.io/kubesphere/pkg/server/params"

	fakeks "kubesphere.io/kubesphere/pkg/client/clientset/versioned/fake"
	"kubesphere.io/kubesphere/pkg/informers"
)

func TestManifest(t *testing.T) {
	manifestOper := prepareManifestOperator()
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "manifest-test"},
	}
	ns, err := k8sClient.CoreV1().Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})
	if err != nil {
		t.Errorf("create test namespace error: %s", err)
	}

	resourceName := "manifest-pj7mf9"
	deploy := &appv1.Deployment{
		TypeMeta: metav1.TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      resourceName,
			Namespace: ns.Name,
		},
	}

	byteDeploy, err := json.Marshal(deploy)
	if err != nil {
		t.Errorf("marshal deployment error: %s", err)
	}
	createManifestReq := CreateManifestRequest{
		Name:             resourceName,
		Description:      "",
		Version:          1,
		AppVersion:       "2.0",
		CustomResource:   string(byteDeploy),
		RelatedResources: nil,
	}

	// create manifest
	err = manifestOper.CreateManifest(testWorkspace, "", ns.Name, createManifestReq)
	if err != nil {
		t.Errorf("create manifest error: %s", err)
		t.FailNow()
	}

	// add manifest to indexer
	manifests, err := ksClient.ApplicationV1alpha1().Manifests().List(context.TODO(), metav1.ListOptions{})
	for _, manifest := range manifests.Items {
		err := fakeInformerFactory.KubeSphereSharedInformerFactory().Application().V1alpha1().Manifests().
			Informer().GetIndexer().Add(&manifest)
		if err != nil {
			klog.Errorf("failed to add manifest to indexer")
			t.FailNow()
		}
	}

	// describe manifest
	_, err = manifestOper.DescribeManifest(testWorkspace, "", ns.Name, resourceName)
	if err != nil {
		t.Errorf("describe manifest error: %s", err)
	}

	// modify manifest
	modifyReq := ModifyManifestRequest{
		Name:           resourceName,
		Description:    "test",
		Version:        2,
		AppVersion:     "2.0",
		CustomResource: string(byteDeploy),
		Workspace:      testWorkspace,
	}
	err = manifestOper.ModifyManifest(modifyReq)
	if err != nil {
		t.Errorf("describe manifest error: %s", err)
	}

	// list manifest
	cond := &params.Conditions{}
	resp, err := manifestOper.ListManifests(testWorkspace, "", ns.Name, cond, 10, 0, "", false)
	if err != nil {
		t.Errorf("list manifest error: %s", err)
		t.FailNow()
	}

	if len(resp.Items) != 1 {
		klog.Errorf("list manifest failed")
		t.FailNow()
	}

	// delete manifest
	err = manifestOper.DeleteManifest(testWorkspace, "", ns.Name, resourceName)
	if err != nil {
		t.Errorf("delete manifest error: %s", err)
		t.FailNow()
	}
}

func prepareManifestOperator() ManifestInterface {
	ksClient = fakeks.NewSimpleClientset()
	k8sClient = fakek8s.NewSimpleClientset()
	fakeInformerFactory = informers.NewInformerFactories(k8sClient, ksClient, nil, nil, nil, nil, nil)
	return newManifestOperator(fakeInformerFactory.KubernetesSharedInformerFactory(), fakeInformerFactory.KubeSphereSharedInformerFactory(), ksClient)
}
