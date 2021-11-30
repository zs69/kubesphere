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

package cert

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"kubesphere.io/kubesphere/pkg/license"

	"k8s.io/klog"
)

type CertificateStore struct {
	Cert *x509.Certificate
}

var CertStore *CertificateStore

func InitCert() error {
	CertStore = &CertificateStore{}
	block, _ := pem.Decode(license.KSCert)
	if block == nil {
		return fmt.Errorf("cert data is empty")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		klog.Infof("decoded ks cert: %s", license.KSCert)
		klog.Errorf("parse certificate failed, error: %s", err)
		return err
	}

	CertStore.Cert = cert
	return nil
}
