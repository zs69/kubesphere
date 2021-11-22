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
	"encoding/base64"
	"encoding/pem"
	"fmt"

	"k8s.io/klog"
)

// KSCert is the public key to verify license signature.
// It's initialized by go build with `-ldflags="-X '${KUBE_GO_PACKAGE}/pkg/simple/client/license/cert.KSCert=$(cat ks-apiserver.pem | base64)'"`
// Please refer to hack/lib/golang.sh#L62-64.
var KSCert string = ""

type CertificateStore struct {
	Cert *x509.Certificate
}

var CertStore *CertificateStore

func InitCert() error {
	CertStore = &CertificateStore{}
	data, err := base64.StdEncoding.DecodeString(KSCert)
	if err != nil {
		klog.Infof("ks cert: %s", KSCert)
		klog.Errorf("decode cert failed, error: %s", err)
		return err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return fmt.Errorf("cert data is empty")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		klog.Infof("decoded ks cert: %s", KSCert)
		klog.Errorf("parse certificate failed, error: %s", err)
		return err
	}

	CertStore.Cert = cert
	return nil
}
