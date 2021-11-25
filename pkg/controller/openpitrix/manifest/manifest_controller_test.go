package manifest

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"kubesphere.io/kubesphere/pkg/utils/idutils"

	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
	//+kubebuilder:scaffold:imports
	"kubesphere.io/api/application/v1alpha1"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var k8sClient client.Client

var _ = Describe("manifest", func() {

	const timeout = time.Second * 360
	const interval = time.Second * 1

	manifest := createManifest()
	BeforeEach(func() {
		err := k8sClient.Create(context.Background(), manifest)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("Manifest Controller", func() {
		It("Should success", func() {
			key := types.NamespacedName{
				Name: manifest.Name,
			}

			By("Expecting manifest state is Created")
			Eventually(func() bool {
				manifest := &v1alpha1.Manifest{}
				_ = k8sClient.Get(context.Background(), key, manifest)
				return manifest.Status.State == v1alpha1.StatusCreated
			}, timeout, interval).Should(BeTrue())
		})
	})

	AfterEach(func() {
		err := k8sClient.Delete(context.Background(), manifest)
		Expect(err).NotTo(HaveOccurred())
	})
})

func createManifest() *v1alpha1.Manifest {
	resourceName := idutils.GetUuid36("")
	namespace := "default"
	appLabel := "manifest-test"
	deploy := &appv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      resourceName,
			Namespace: namespace,
		},
		Spec: appv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": appLabel,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": appLabel,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            appLabel,
							Image:           "nginx:1.13.5-alpine",
							ImagePullPolicy: "IfNotPresent",
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									Protocol:      corev1.ProtocolTCP,
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}
	byteDeploy, err := json.Marshal(deploy)
	if err != nil {
		fmt.Printf("marshal error: %s", err)
	}
	return &v1alpha1.Manifest{
		ObjectMeta: metav1.ObjectMeta{
			Name: resourceName,
		},
		Spec: v1alpha1.ManifestSpec{
			Cluster:        "host",
			Namespace:      "default",
			AppVersion:     "2.0",
			CustomResource: string(byteDeploy),
			Version:        1,
		},
	}
}

func int32Ptr(i int32) *int32 {
	return &i
}
