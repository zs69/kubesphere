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

var LicenseData = `{"l_id":"16rl73m9k94k43","l_type":"sub","ver":1,"subj":{"co":"","name":"lihui"},"issuer":{"co":"qingcloud","name":"qingcloud"},"not_before":"2021-07-22T00:00:00Z","not_after":"2024-12-23T00:00:00Z","issue_at":"2021-11-22T12:24:56.197621Z","max_cluster":1,"max_node":3,"max_core":5,"sig":"jOw9ZiXCXiyyGJbSxy2IzyCbngFQDl7r4UnXmA2NqZalsKdyShdgEcESGM5BeaJMAV0RoRj5dbaaWY2Y5x5tEn+aVibRcmCLOH6gjmvA/M57NuaCc3AxQUQYhIM8xaiQJcDqI2vWZ+S9q494n3tjIS7zrzbd5Esd61R0NCS0Vhc+iaAnAnIOwwA8Osi4L5UNbsUhxuK4XNolo60poB7kKB1sJj11vqf4zzXQLmqFcBxSo0TNTQdu/9TyO0QawQFBeAoayZ0Qunisx7kjs431PpMMCpYLpuFS5bPaMIklo0121Q2bGztRlw4i1d4QZXBpKheL3jBZmQWbm6zqyxar4CGeqPQ4Z/UIRBZuMBGFZ0rw5qVZ4puRk6nKK9jYh74gtoyKy6eM2gfh9XgNyZOnRxu9t5zV80vEYQTLvELZz98kv+c/pLxkd+CW/QarYqSpQDwjHGYbQ3PP1YMgKhuRwvHlaSE8FsoMY045YuSiGYCvYVtNqDOQYF4gUwBdmeMA1vA1+BIQMxBnV8/dZKQyw/415edGksDyo0jdfYLor93+NtQIDhePqSEzrC8wKhLCOORJ7/uiIUPZGb54P91xGR+iNqSJdMgURfHGvrEDXLynLrOZqhafsxkLfehb3EIY8O8EANJxlwiQ3OyjAXbCfZUlYKG03Wqrv3hcU8K9vxM="}`
var licenseSecret *v1.Secret
var _ = Describe("license_controller", func() {
	cert.KSCert = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUdxakNDQkpLZ0F3SUJBZ0lVVzQ0eGQ1UVBRclFuVFlVdmRtU0tHTS9LYWhJd0RRWUpLb1pJaHZjTkFRRU4KQlFBd1l6RUxNQWtHQTFVRUJoTUNRMDR4Q3pBSkJnTlZCQWdUQWtoQ01Rc3dDUVlEVlFRSEV3SlhTREVTTUJBRwpBMVVFQ2hNSlVXbHVaME5zYjNWa01STXdFUVlEVlFRTEV3cExkV0psYzNCb1pYSmxNUkV3RHdZRFZRUURFd2hyCmN5MWpiRzkxWkRBZUZ3MHlNVEEzTVRrd09UVTNNREJhRncweU1qQTNNVGt3T1RVM01EQmFNR2N4Q3pBSkJnTlYKQkFZVEFrTk9NUXN3Q1FZRFZRUUlFd0pJUWpFTE1Ba0dBMVVFQnhNQ1YwZ3hFakFRQmdOVkJBb1RDVkZwYm1kRApiRzkxWkRFVE1CRUdBMVVFQ3hNS1MzVmlaWE53YUdWeVpURVZNQk1HQTFVRUF4TU1hM010WVhCcGMyVnlkbVZ5Ck1JSUNJakFOQmdrcWhraUc5dzBCQVFFRkFBT0NBZzhBTUlJQ0NnS0NBZ0VBeFIyNEdmMHlFYXh2VDlFVjZWV1gKelhNTlp5OEg1U25nTzlTUnk0d3FZR1k2RjNhZG9EakhGRnZGNWZDZEl0S0FwSUo3MHd0MGEvRHJLMXVadGRGVQpDVklBZzVzRFBYU1d3U3lxWXllT29WbC9oN2I2SlVVZ0tFeUJ0MWxyZG9HTnNCT3QvM0xvU2lqM1hSYzlBcGZJClRTWUluYmxQaW81Sm5KUHcvaFJvVTkrNXJqUGF1am9VM0lMRnJCalo2aEU2MzNVb25MemdmNkd0dlBnbEtweWUKR1dUSHU1S0IyQkgyL0xPSlFpaWlOSVRPNGhhRW0xcDN6WVpRY1ZSbTJWa0JZaHpUZkJDdzd6ZlVhaERyc3lReQpjWVJ2V1ZuVEJhTVAxNXlOaEJWY0kvUnlKRCszVmNJVUM5TjdndjZZRWQ3NnREN0xPTzZFMVdSWHhEcm43OGtFCnFzYSs2RlNUbm4yTzlXNjN0emR5bjlVUkI3YUVmeTFOcXNjTFo0MFo2SW4zOUNNUnVJY0lneXFpMmc1YVNzTHIKWnBGb1pwbm45QmxxRnlab2NlcnZNUXJFRUlIaDJBS0VPMTlzVGhBTWNVNndXcjZneHNxcWFrUTh4OHowcVFucgpVUGlURER2Y2ZGemxjeTBpS0lEbWFVbGRiL29UdkVHQmcvc0FZbG81Wmg5cWExY3Q5bUluVUdjZ2FyeTl3UFhpCk91MWJva3k0VDNzM0tiQWlYeDBaRkVTVm1oTk02dmkvNUYwUDFRR0JsbjJUUjVUVjNvdGJIMlJTcHJEaVN6UEQKMnRYNXNKWXIzbGszaFhoZFdMc3FKczFKZWNlUGxySkJpUm9YdnB4cGtTUFgrUTYrQWorYjhJSVZEMVhOOTNuQQovR1g5UW5FcGR4ZVdJVXB6OUEwNGx3RUNBd0VBQWFPQ0FWQXdnZ0ZNTUE0R0ExVWREd0VCL3dRRUF3SUZvREFkCkJnTlZIU1VFRmpBVUJnZ3JCZ0VGQlFjREFRWUlLd1lCQlFVSEF3SXdEQVlEVlIwVEFRSC9CQUl3QURBZEJnTlYKSFE0RUZnUVUycFZaRnFMRUlyMFYzQ2x0YllWVEowK2V6amN3SHdZRFZSMGpCQmd3Rm9BVWIwSitnczRzZkNVNwpkQyt2ak40dW5xZlhuWnN3Z2N3R0ExVWRFUVNCeERDQndZSUpiRzlqWVd4b2IzTjBnZ3hyY3kxaGNHbHpaWEoyClpYS0NIbXR6TFdGd2FYTmxjblpsY2k1cmRXSmxjM0JvWlhKbExYTjVjM1JsYllJaWEzTXRZWEJwYzJWeWRtVnkKTG10MVltVnpjR2hsY21VdGMzbHpkR1Z0TG5OMlk0SXFhM010WVhCcGMyVnlkbVZ5TG10MVltVnpjR2hsY21VdApjM2x6ZEdWdExuTjJZeTVqYkhWemRHVnlnakJyY3kxaGNHbHpaWEoyWlhJdWEzVmlaWE53YUdWeVpTMXplWE4wClpXMHVjM1pqTG1Oc2RYTjBaWEl1Ykc5allXeUhCSDhBQUFFd0RRWUpLb1pJaHZjTkFRRU5CUUFEZ2dJQkFMN1MKYytKK3YxMEVtOW90TEovdmlwYmZkdnFMcC9FSklZVjRIQ3kwb2NRYThsUzB1cnV0c0l4Mm1WMzIwS3dJWGcvUgpqUmJSdUJsSTJyY29pMzNXUW9ROGUvaWE3YXdXWTlTWGkyZG5NSXkxSjROaDg1M2ZNYitNNGpiZUQvcnVCV3dHCkk0L2o2cmlyTWw1Snk1Wlc2U2g0STE3YWtKVEczalMwbVptRHBITmVZVVhCTEYxNmRVRXB0eUN0WTh0RkJIYzMKZk5PRy9aZUJOUXVOa1FHVk5LSU9jemRWcGZTaU1hbUVWOUVKWkJaM3k4UXVLVGEvR0tobS9UdjFTQk1pMXVrZwpjbEh4bm8yY2dHeDZEOTNlZmgwdXdrdWNHOTJCMlhQUXhBb0w4bUNxR3Y0Y1FYYTFCdE9aSk9qeTJxQkJKSTZZCktKbG9FeUFDbGNUTTQyczhEZjhGWGlodVN6WE1jbDUvelBkdUgrUnJBNkx0ak04REFOa2N1WEtOWGVUZEVoS3YKTVdmY1NLWXdoOU4vNVcyWVlYK1F5czB5cmd0Um9Ic3FMOGFJTktKS2VaZmdSQ3VlbmFjM2d6VjhRUzRhQnRrTwpwTGFBQjRLREF6MndqejBsekhjYnFoOXZTMDNaMnhRU2tVUFBFd3R4VzZMM3YwYjRVRnhtQldrOEE4NzdlWEI3CmFaSW4zOUFuSEFibE5IZ1RpV2RIa05MWG9CckpmQnZRdlFFNFczT0lWWTViREQ1ZXR2WVhnMklhN2c5WVBhQ1UKd0Zwc0s4VHBTSFozV3JFbW95V0J0UzFxbHlGS29iVzZyeVBFM2pIYXh1eUV6R2wvZmpvQnYxbTI4SkcxVDA1UAplbVJUdEJLS2MyeEEvbGR1YmMzQ0FSNHFKL3BSMUx2Zkg4STZOTjhsCi0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K"
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
		It("license is valid, should success", func() {
			By("create 3 nodes")
			for _, name := range []string{"node1", "node2", "node3"} {
				err := k8sClient.Create(context.Background(), &v1.Node{ObjectMeta: v12.ObjectMeta{
					Name: name,
				}})
				Expect(err).NotTo(HaveOccurred())
			}

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

	Context("node count limit exceeded", func() {
		BeforeEach(func() {
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

		It("Should success", func() {
			By("create one node")
			for _, name := range []string{"node4"} {
				err := k8sClient.Create(context.Background(), &v1.Node{ObjectMeta: v12.ObjectMeta{
					Name: name,
				}})
				Expect(err).NotTo(HaveOccurred())
			}

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
					return ls.Violation.Type == licensetypes.NodeOverflow
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

		It("Should success", func() {
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
