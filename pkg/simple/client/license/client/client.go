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

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"kubesphere.io/kubesphere/pkg/simple/client/license/cert"
	licensetypes "kubesphere.io/kubesphere/pkg/simple/client/license/types/v1alpha1"
)

const (
	LicensePath        = "/kapis/license.kubesphere.io/v1alpha1/license/ks-license"
	DefaultKSApiserver = "https://ks-apiserver.kubesphere-system"
)

type License struct {
	Data   *licensetypes.License       `json:"data"`
	Status *licensetypes.LicenseStatus `json:"status"`
}

type LicenseClient struct {
	apiserver string
	certStore *cert.CertificateStore
}

// NewLicenseClient create a client to get license info from ks-apiserver
// apiserver the server to connect, default value is https://ks-apiserver.kubesphere-system
func NewLicenseClient(apiserver string) *LicenseClient {
	if apiserver == "" {
		apiserver = DefaultKSApiserver
	}

	return &LicenseClient{
		apiserver: apiserver,
		certStore: cert.CertStore,
	}
}

func (lc *LicenseClient) GetLicense(ctx context.Context) (*License, error) {
	client := &http.Client{
		Transport: &http.Transport{},
	}

	var url string
	if strings.HasSuffix(lc.apiserver, "/") {
		url = fmt.Sprintf("%s/%s", lc.apiserver, LicensePath)
	} else {
		url = fmt.Sprintf("%s%s", lc.apiserver, LicensePath)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	licenseRsp := &License{}

	err = json.Unmarshal(body, licenseRsp)

	return licenseRsp, err
}
