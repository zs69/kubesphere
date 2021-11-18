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

package v1alpha1

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"time"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v12 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/klog"

	"kubesphere.io/kubesphere/pkg/constants"
)

const (
	NoViolation        = "no_violation"
	EmptyLicense       = "empty_license"
	FormatError        = "format_error"
	InvalidSignature   = "invalid_signature"
	TimeExpired        = "time_expired"
	TimeNotStart       = "time_not_start"
	VersionNotMatch    = "version_not_match"
	NodeOverflow       = "node_overflow"
	CpuOverflow        = "cpu_overflow"
	CoreOverflow       = "core_overflow"
	ClusterOverflow    = "cluster_overflow"
	ClusterNotMatch    = "cluster_not_match"
	InvalidLicenseType = "invalid_type"

	LicenseName = "ks-license"
	LicenseKey  = "license"

	LicenseStatusKey = "license.kubesphere.io/status"

	LicenseTypeSub         = "sub"
	LicenseTypeManged      = "managed"
	LicenseTypeMaintenance = "ma"

	LicenseViolationCode = 430
)

type Violation struct {
	Component string `json:"component"`
	Type      string `json:"type"`

	Reason string `json:"reason,omitempty"`
	// current value of node or vm count
	Current int `json:"current,omitempty"`
	// the expected value
	Expected int `json:"expected,omitempty"`

	EndTime   *time.Time `json:"end_time,omitempty"`
	StartTime *time.Time `json:"start_time,omitempty"`
}

type ClusterInfo struct {
	CPUNum  int `json:"cpu_num,omitempty"`
	CoreNum int `json:"core_num,omitempty"`
	NodeNum int `json:"node_num,omitempty"`
}

type LicenseStatus struct {
	Host       ClusterInfo `json:"host,omitempty"`
	Member     ClusterInfo `json:"member,omitempty"`
	ClusterNum int         `json:"cluster_num,omitempty"`

	Violation Violation `json:"violation"`
}

type License struct {
	// license id which uniquely identifies the license. It's MUST NOT be empty.
	LicenseId string `json:"l_id,omitempty"`
	ClusterId string `json:"c_id,omitempty"`
	// the type of the license, valid values are `ma` for maintenance, `sub` for subscription, and `managed` from managed k8s.
	LicenseType string `json:"l_type"`
	// license version number.
	Version int `json:"ver,omitempty"`
	// The user who will use this license.
	Subject User `json:"subj,omitempty"`
	// The issuer who issued this license.
	Issuer User `json:"issuer,omitempty"`
	// License is not valid before this time
	NotBefore *time.Time `json:"not_before,omitempty"`
	// License is not valid after this time.
	NotAfter *time.Time `json:"not_after,omitempty"`
	// The end time of maintenance.
	MaintenanceEnd *time.Time `json:"ma_end,omitempty"`
	// license issue time
	IssueAt time.Time `json:"issue_at,omitempty"`
	// Max clusters for this license.
	MaxCluster int `json:"max_cluster,omitempty"`
	// Max Node for this license.
	MaxNode int `json:"max_node,omitempty"`
	// Max cpu num for this license.
	MaxCPU int `json:"max_cpu,omitempty"`
	// Max CPU Core for this license.
	MaxCore int `json:"max_core,omitempty"`
	// Max Virtual Machine for this license.
	MaxVM       int `json:"max_vm,omitempty"`
	GracePeriod int `json:"grace_period,omitempty"`
	// ks-controller-manager must be in the range of [start_version, end_version)
	StartVersion         string       `json:"start_ver,omitempty"`
	EndVersion           string       `json:"end_ver,omitempty"`
	ComponentConstraints []Constraint `json:"component_constraints,omitempty"`

	// ID to identify the client
	APIKey string `json:"api_key,omitempty"`
	// Secret to connect to kubesphere cloud
	APISecret string `json:"api_secret,omitempty"`
	// An endpoint from where to fetch new license
	APIEndpoint string `json:"api_ep,omitempty"`

	Signature string `json:"sig"`
}

type User struct {
	Corporation string `json:"co"`
	Name        string `json:"name,omitempty"`
	Id          string `json:"id,omitempty"`
}

type Constraint struct {
	Name string `json:"name"`
	// constraint type
	Type      string     `json:"type"`
	Value     string     `json:"value"`
	NotAfter  *time.Time `json:"not_after,omitempty"`
	NotBefore *time.Time `json:"not_before,omitempty"`
}

func (l *License) IsExpired() (bool, *Violation) {
	now := time.Now().UTC()

	if l.NotBefore != nil {
		if now.Before(*l.NotBefore) {
			return true, &Violation{Type: TimeNotStart, StartTime: l.NotBefore, EndTime: l.NotAfter}
		}
	}

	if l.NotAfter != nil {
		if now.After(*l.NotAfter) {
			return true, &Violation{Type: TimeExpired, StartTime: l.NotBefore, EndTime: l.NotAfter}
		}
	}

	for _, cc := range l.ComponentConstraints {
		if cc.NotBefore != nil && now.Before(*cc.NotBefore) {
			return true, &Violation{Type: TimeNotStart, StartTime: cc.NotBefore, EndTime: cc.NotAfter}
		}
		if cc.NotAfter != nil && now.After(*cc.NotAfter) {
			return true, &Violation{Type: TimeExpired, StartTime: cc.NotBefore, EndTime: cc.NotAfter}
		}
	}

	return false, nil
}

func (l *License) IsEmpty() bool {
	if l.LicenseId == "" {
		return true
	}
	return false
}

// LoadLicense unmarshals the data
// If it's a valid license, return then license, if it's not a valid license, return en empty license
// data: the license data
func LoadLicense(data []byte) (*License, error) {
	l := &License{}

	err := json.Unmarshal(data, l)

	return l, err
}

// Verify verify the signature of the license
func (l *License) Verify(cert *x509.Certificate) (bool, *Violation) {
	// Create a new license, because we need overwrite signature when verify it.
	newLicense := *l

	rsaPublicKey := cert.PublicKey.(*rsa.PublicKey)
	decodedSignature, _ := base64.StdEncoding.DecodeString(newLicense.Signature)

	newLicense.Signature = ""
	data, _ := json.Marshal(newLicense)
	digest := sha256.Sum256(data)
	verifyErr := rsa.VerifyPKCS1v15(rsaPublicKey, crypto.SHA256, digest[:], decodedSignature)

	if verifyErr != nil {
		return false, &Violation{
			Type:   InvalidSignature,
			Reason: verifyErr.Error(),
		}
	}

	return true, nil
}

// Sign add signature to the license
func (l *License) Sign(key []byte) (err error) {
	block, _ := pem.Decode(key)
	if block == nil {
		return errors.New("failed to parse PEM block containing the key")
	}

	priKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil
	}
	l.Signature = ""
	data, _ := json.Marshal(l)
	digest := sha256.Sum256(data)

	signature, signErr := rsa.SignPKCS1v15(rand.Reader, priKey, crypto.SHA256, digest[:])

	if signErr != nil {
		return err
	}

	// just to check that we can survive to and from b64
	b64sig := base64.StdEncoding.EncodeToString(signature)
	l.Signature = b64sig
	return nil
}

// SaveLicenseData save license to secret
func (l *License) SaveLicenseData(secretInterface v12.SecretInterface) (err error) {
	oldSecret, err := secretInterface.Get(context.TODO(), LicenseName, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	data, _ := json.Marshal(l)

	if apierrors.IsNotFound(err) {
		secret := v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      LicenseName,
				Namespace: constants.KubeSphereNamespace,
			},
			Data: map[string][]byte{
				LicenseKey: data,
			},
		}

		_, err = secretInterface.Create(context.Background(), &secret, metav1.CreateOptions{})
		if err == nil {
			klog.V(2).Infof("license created")
		} else {
			klog.Errorf("create license data failed: %s", err)
		}
	} else {
		secret := oldSecret.DeepCopy()
		secret.Data[LicenseKey] = data

		// Update old secret
		_, err = secretInterface.Update(context.Background(), secret, metav1.UpdateOptions{})
		if err == nil {
			klog.V(2).Infof("license updated")
		} else {
			klog.Errorf("update license data failed: %s", err)
		}
	}

	return err
}

func (l *License) Check(cert *x509.Certificate, cid string, checker ...Checker) (*Violation, error) {
	if l.IsEmpty() {
		return &Violation{Type: EmptyLicense}, nil
	}

	if _, vio := l.Verify(cert); vio != nil {
		return vio, nil
	}

	if expired, vio := l.IsExpired(); expired {
		return vio, nil
	}

	for _, c := range checker {
		vio, err := c.Check(l)
		if err != nil {
			return nil, err
		}
		if vio != nil {
			return vio, nil
		}
	}

	return nil, nil
}

type Checker interface {
	Check(l *License) (*Violation, error)
}
