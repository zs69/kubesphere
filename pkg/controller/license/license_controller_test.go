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

package license_test

import (
	"context"
	"encoding/json"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"

	"kubesphere.io/kubesphere/pkg/license"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"kubesphere.io/kubesphere/pkg/constants"
	"kubesphere.io/kubesphere/pkg/simple/client/license/cert"
	licensetypes "kubesphere.io/kubesphere/pkg/simple/client/license/types/v1alpha1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var LicenseData = `{"licenseId":"44n6mnv6wqm17n","licenseType":"subscription","version":1,"subject":{"co":"","name":"lihui"},"issuer":{"co":"qingcloud","name":"qingcloud"},"notBefore":"2021-07-22T00:00:00Z","notAfter":"2044-05-15T08:00:00Z","issueAt":"2021-11-23T11:55:32.793746Z","maxCluster":1,"maxNode":3,"maxCore":5,"signature":"ICsYi8TkRW6JZ4a5T8Fu62aDo7HZ4ekknz1yavYAOpX5r1cqI9IZGY1asHgUvPb1LfwMLT/Ej3ermR7bOLOhhAPYMiLZubu39BLCIFYWwNK8cHlxi999jyfxMlTA4zyDVsMEu3c6PDGXwu+CZQaCyoqzturHUcrazAxh7v2EX3rhkVvQuILU9fQzIDG6lLwmUZLQ3G0ckwQ90C1ImAMKFSBi1AUXOA6VT9eCPtShJCGfqSP4hqAYH1Zb+ZmWKakNfffKgw7Tlmeymclu2xeVTADmRYoGsGttIAGaXFLXOx9e8q8X26vnnYXGLvWurkXPE8itvVUotM2LomfvBca2lUDyleHiQSkcFdlJEQ55ithE67w8bdg+MuIejbsYpzIjFhewCuUCeqQKIxE2YVQlUic07K8YudVgfMJA09vCBaZcbUENrRK4KTI0zjAHWAx6OjU9d1EUTAPfxD9gRQswMEg1XUQmASqUIFR20i+rC2U3pdFRHxZk2Xh2pFTXXldzDplh1T7ftBGDQyz5mou0kX8zuxbIcC/kYT7QLh80+A42EzILzG7jcR5hTrUiWizKjsyP5TTqxUwjPo9bXMmyURsoD5wMiQ2hIPWTPwOjygCt/6LsR5kzqVQQznqEn3RQOVqTZ4UZPOMcWW8GLqytPHe555IS8N0KrH32laZyIx4="}`
var licenseSecret *v1.Secret
var _ = Describe("license_controller", func() {
	license.KSCert = []byte("-----BEGIN CERTIFICATE-----\nMIIGqjCCBJKgAwIBAgIUW44xd5QPQrQnTYUvdmSKGM/KahIwDQYJKoZIhvcNAQEN\nBQAwYzELMAkGA1UEBhMCQ04xCzAJBgNVBAgTAkhCMQswCQYDVQQHEwJXSDESMBAG\nA1UEChMJUWluZ0Nsb3VkMRMwEQYDVQQLEwpLdWJlc3BoZXJlMREwDwYDVQQDEwhr\ncy1jbG91ZDAeFw0yMTA3MTkwOTU3MDBaFw0yMjA3MTkwOTU3MDBaMGcxCzAJBgNV\nBAYTAkNOMQswCQYDVQQIEwJIQjELMAkGA1UEBxMCV0gxEjAQBgNVBAoTCVFpbmdD\nbG91ZDETMBEGA1UECxMKS3ViZXNwaGVyZTEVMBMGA1UEAxMMa3MtYXBpc2VydmVy\nMIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAxR24Gf0yEaxvT9EV6VWX\nzXMNZy8H5SngO9SRy4wqYGY6F3adoDjHFFvF5fCdItKApIJ70wt0a/DrK1uZtdFU\nCVIAg5sDPXSWwSyqYyeOoVl/h7b6JUUgKEyBt1lrdoGNsBOt/3LoSij3XRc9ApfI\nTSYInblPio5JnJPw/hRoU9+5rjPaujoU3ILFrBjZ6hE633UonLzgf6GtvPglKpye\nGWTHu5KB2BH2/LOJQiiiNITO4haEm1p3zYZQcVRm2VkBYhzTfBCw7zfUahDrsyQy\ncYRvWVnTBaMP15yNhBVcI/RyJD+3VcIUC9N7gv6YEd76tD7LOO6E1WRXxDrn78kE\nqsa+6FSTnn2O9W63tzdyn9URB7aEfy1NqscLZ40Z6In39CMRuIcIgyqi2g5aSsLr\nZpFoZpnn9BlqFyZocervMQrEEIHh2AKEO19sThAMcU6wWr6gxsqqakQ8x8z0qQnr\nUPiTDDvcfFzlcy0iKIDmaUldb/oTvEGBg/sAYlo5Zh9qa1ct9mInUGcgary9wPXi\nOu1boky4T3s3KbAiXx0ZFESVmhNM6vi/5F0P1QGBln2TR5TV3otbH2RSprDiSzPD\n2tX5sJYr3lk3hXhdWLsqJs1JecePlrJBiRoXvpxpkSPX+Q6+Aj+b8IIVD1XN93nA\n/GX9QnEpdxeWIUpz9A04lwECAwEAAaOCAVAwggFMMA4GA1UdDwEB/wQEAwIFoDAd\nBgNVHSUEFjAUBggrBgEFBQcDAQYIKwYBBQUHAwIwDAYDVR0TAQH/BAIwADAdBgNV\nHQ4EFgQU2pVZFqLEIr0V3CltbYVTJ0+ezjcwHwYDVR0jBBgwFoAUb0J+gs4sfCU7\ndC+vjN4unqfXnZswgcwGA1UdEQSBxDCBwYIJbG9jYWxob3N0ggxrcy1hcGlzZXJ2\nZXKCHmtzLWFwaXNlcnZlci5rdWJlc3BoZXJlLXN5c3RlbYIia3MtYXBpc2VydmVy\nLmt1YmVzcGhlcmUtc3lzdGVtLnN2Y4Iqa3MtYXBpc2VydmVyLmt1YmVzcGhlcmUt\nc3lzdGVtLnN2Yy5jbHVzdGVygjBrcy1hcGlzZXJ2ZXIua3ViZXNwaGVyZS1zeXN0\nZW0uc3ZjLmNsdXN0ZXIubG9jYWyHBH8AAAEwDQYJKoZIhvcNAQENBQADggIBAL7S\nc+J+v10Em9otLJ/vipbfdvqLp/EJIYV4HCy0ocQa8lS0urutsIx2mV320KwIXg/R\njRbRuBlI2rcoi33WQoQ8e/ia7awWY9SXi2dnMIy1J4Nh853fMb+M4jbeD/ruBWwG\nI4/j6rirMl5Jy5ZW6Sh4I17akJTG3jS0mZmDpHNeYUXBLF16dUEptyCtY8tFBHc3\nfNOG/ZeBNQuNkQGVNKIOczdVpfSiMamEV9EJZBZ3y8QuKTa/GKhm/Tv1SBMi1ukg\nclHxno2cgGx6D93efh0uwkucG92B2XPQxAoL8mCqGv4cQXa1BtOZJOjy2qBBJI6Y\nKJloEyAClcTM42s8Df8FXihuSzXMcl5/zPduH+RrA6LtjM8DANkcuXKNXeTdEhKv\nMWfcSKYwh9N/5W2YYX+Qys0yrgtRoHsqL8aINKJKeZfgRCuenac3gzV8QS4aBtkO\npLaAB4KDAz2wjz0lzHcbqh9vS03Z2xQSkUPPEwtxW6L3v0b4UFxmBWk8A877eXB7\naZIn39AnHAblNHgTiWdHkNLXoBrJfBvQvQE4W3OIVY5bDD5etvYXg2Ia7g9YPaCU\nwFpsK8TpSHZ3WrEmoyWBtS1qlyFKobW6ryPE3jHaxuyEzGl/fjoBv1m28JG1T05P\nemRTtBKKc2xA/ldubc3CAR4qJ/pR1LvfH8I6NN8l\n-----END CERTIFICATE-----\n")
	cert.InitCert()
	const timeout = time.Second * 240
	const interval = time.Second * 1
	licenseSecret = &v1.Secret{ObjectMeta: v12.ObjectMeta{
		Name:      licensetypes.LicenseName,
		Namespace: constants.KubeSphereNamespace,
	}, Data: map[string][]byte{
		licensetypes.LicenseKey: []byte(LicenseData)},
	}

	Context("license is valid", func() {
		BeforeEach(func() {
			By("create 1 nodes")
			for _, name := range []string{"node1"} {
				err := k8sClient.Create(context.Background(), &v1.Node{ObjectMeta: v12.ObjectMeta{
					Name: name,
				}})
				Expect(err).NotTo(HaveOccurred())
			}

			ns := v1.Namespace{}
			err := k8sClient.Get(context.Background(), types.NamespacedName{Name: constants.KubeSphereNamespace}, &ns)
			if apierrors.IsNotFound(err) {
				err = k8sClient.Create(context.Background(), &v1.Namespace{ObjectMeta: v12.ObjectMeta{Name: constants.KubeSphereNamespace}})
			} else {
				Expect(err).NotTo(HaveOccurred())
			}
			// create license data
			err = k8sClient.Create(context.Background(), licenseSecret.DeepCopy())
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			err := k8sClient.Delete(context.Background(), &v1.Secret{ObjectMeta: v12.ObjectMeta{Namespace: licenseSecret.Namespace,
				Name: licenseSecret.Name}})
			Expect(err).NotTo(HaveOccurred())
		})
		It("should success", func() {
			Eventually(func() bool {
				secret := &v1.Secret{}
				k8sClient.Get(context.Background(),
					types.NamespacedName{Name: licensetypes.LicenseName, Namespace: constants.KubeSphereNamespace}, secret)
				status := secret.Annotations[licensetypes.LicenseStatusKey]
				if len(status) == 0 {
					return false
				} else {
					ls := licensetypes.LicenseStatus{}
					err := json.Unmarshal([]byte(status), &ls)
					Expect(err).NotTo(HaveOccurred())
					return ls.Violation.Type == licensetypes.NoViolation
				}

			}, timeout, interval).Should(BeTrue())

		})
	})

	Context("core count limit exceeded", func() {
		BeforeEach(func() {
			By("create one node")
			cpuQuantity := resource.NewQuantity(6, resource.DecimalSI)
			for _, name := range []string{"node2"} {
				err := k8sClient.Create(context.Background(), &v1.Node{ObjectMeta: v12.ObjectMeta{
					Name: name,
				}, Status: v1.NodeStatus{
					Capacity: map[v1.ResourceName]resource.Quantity{
						v1.ResourceCPU: *cpuQuantity,
					},
				},
				})
				Expect(err).NotTo(HaveOccurred())
			}
			ns := v1.Namespace{}
			err := k8sClient.Get(context.Background(), types.NamespacedName{Name: constants.KubeSphereNamespace}, &ns)
			if apierrors.IsNotFound(err) {
				err = k8sClient.Create(context.Background(), &v1.Namespace{ObjectMeta: v12.ObjectMeta{Name: constants.KubeSphereNamespace}})
			} else {
				Expect(err).NotTo(HaveOccurred())
			}
			err = k8sClient.Create(context.Background(), licenseSecret.DeepCopy())
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			err := k8sClient.Delete(context.Background(), &v1.Secret{ObjectMeta: v12.ObjectMeta{Namespace: licenseSecret.Namespace,
				Name: licenseSecret.Name}})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should success", func() {
			Eventually(func() bool {
				secret := &v1.Secret{}
				k8sClient.Get(context.Background(),
					types.NamespacedName{Name: licensetypes.LicenseName, Namespace: constants.KubeSphereNamespace}, secret)
				status := secret.Annotations[licensetypes.LicenseStatusKey]
				if len(status) == 0 {
					return false
				} else {
					ls := licensetypes.LicenseStatus{}
					err := json.Unmarshal([]byte(status), &ls)
					Expect(err).NotTo(HaveOccurred())
					return ls.Violation.Type == licensetypes.CoreCountLimitExceeded
				}

			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("license is empty", func() {
		BeforeEach(func() {
			ns := v1.Namespace{}
			err := k8sClient.Get(context.Background(), types.NamespacedName{Name: constants.KubeSphereNamespace}, &ns)
			if apierrors.IsNotFound(err) {
				err = k8sClient.Create(context.Background(), &v1.Namespace{ObjectMeta: v12.ObjectMeta{Name: constants.KubeSphereNamespace}})
			} else {
				Expect(err).NotTo(HaveOccurred())
			}

			secret := licenseSecret.DeepCopy()
			secret.Data = map[string][]byte{}
			err = k8sClient.Create(context.Background(), secret)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			err := k8sClient.Delete(context.Background(), &v1.Secret{ObjectMeta: v12.ObjectMeta{Namespace: licenseSecret.Namespace,
				Name: licenseSecret.Name}})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should success", func() {
			Eventually(func() bool {
				secret := &v1.Secret{}
				k8sClient.Get(context.Background(),
					types.NamespacedName{Name: licensetypes.LicenseName, Namespace: constants.KubeSphereNamespace}, secret)
				status := secret.Annotations[licensetypes.LicenseStatusKey]
				if len(status) == 0 {
					return false
				} else {
					ls := licensetypes.LicenseStatus{}
					err := json.Unmarshal([]byte(status), &ls)
					Expect(err).NotTo(HaveOccurred())
					return ls.Violation.Type == licensetypes.EmptyLicense
				}

			}, timeout, interval).Should(BeTrue())
		})
	})
})
