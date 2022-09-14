/*
 * Copyright 2022 The KubeSphere Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package helmcmdrelease

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/ioutil"
	"strconv"

	yaml "gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/release"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"kubesphere.io/api/application/v1alpha1"

	"kubesphere.io/kubesphere/pkg/constants"
)

var (
	magicGzip         = []byte{0x1f, 0x8b, 0x08}
	ErrNotHelmRelease = errors.New("not helm release")
	releseStatusMap   = map[release.Status]string{
		release.StatusUnknown:         v1alpha1.HelmStatusCreating,
		release.StatusDeployed:        v1alpha1.HelmStatusActive,
		release.StatusPendingInstall:  v1alpha1.HelmStatusCreated,
		release.StatusUninstalled:     v1alpha1.HelmStatusDeleting,
		release.StatusUninstalling:    v1alpha1.HelmStatusDeleting,
		release.StatusFailed:          v1alpha1.HelmStatusFailed,
		release.StatusPendingUpgrade:  v1alpha1.HelmStatusUpgrading,
		release.StatusPendingRollback: v1alpha1.HelmStatusRollbacking,
		release.StatusSuperseded:      "",
	}
)

func isHelm3Release(labels map[string]string) bool {
	return labels["owner"] == "helm" && labels["name"] != ""
}

func ReleaseVersion(obj runtime.Object) (int, error) {
	m, err := meta.Accessor(obj)
	if err != nil {
		return 0, err
	}
	labels := m.GetLabels()

	version := labels["version"]
	if version == "" {
		version = labels["VERSION"]
	}

	return strconv.Atoi(version)
}

// ToReleaseCR create a helm release from configmap or secret object.
func ToReleaseCR(obj runtime.Object) (*v1alpha1.HelmRelease, error) {
	m, err := meta.Accessor(obj)
	if err != nil {
		return nil, err
	}
	if !isHelm3Release(m.GetLabels()) {
		return nil, ErrNotHelmRelease
	}

	data, err := getReleaseData(obj)
	if err != nil {
		return nil, err
	}

	return buildCRFromHelm3Release(m.GetNamespace(), data)
}

func buildCRFromHelm3Release(namespace, data string) (*v1alpha1.HelmRelease, error) {
	helm3Release, err := decodeHelm3Release(data)
	if err != nil {
		return nil, err
	}

	rlsCR := &v1alpha1.HelmRelease{
		TypeMeta: v1.TypeMeta{
			Kind:       v1alpha1.ResourceKindHelmRelease,
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
		},
		ObjectMeta: v1.ObjectMeta{
			Labels: map[string]string{
				constants.NamespaceLabelKey: namespace,
			},
			Annotations: map[string]string{
				constants.CreatorAnnotationKey: constants.AdminUserName,
			},
			CreationTimestamp: v1.Time{Time: helm3Release.Info.FirstDeployed.Time},
		},
	}

	if helm3Release.Chart != nil {
		if helm3Release.Chart.Metadata != nil {
			rlsCR.Spec.ChartName = helm3Release.Chart.Metadata.Name
			rlsCR.Spec.ChartVersion = helm3Release.Chart.Metadata.Version
			rlsCR.Spec.ChartAppVersion = helm3Release.Chart.Metadata.AppVersion
		}
	}

	rlsCR.Spec.Version = helm3Release.Version
	rlsCR.Spec.Name = helm3Release.Name

	if helm3Release.Config != nil {
		values, _ := yaml.Marshal(helm3Release.Config)
		rlsCR.Spec.Values = values
	} else {
		values, _ := yaml.Marshal(helm3Release.Chart.Values)
		rlsCR.Spec.Values = values
	}

	rlsCR.Status.LastDeployed = &v1.Time{Time: helm3Release.Info.LastDeployed.Time}
	// Map status of helm cmd to status of release cr
	rlsCR.Status.State = releseStatusMap[helm3Release.Info.Status]
	rlsCR.Status.Version = helm3Release.Version

	return rlsCR, err
}

func getReleaseData(obj runtime.Object) (string, error) {
	switch t := obj.(type) {
	case *corev1.Secret:
		return string(t.Data["release"]), nil
	case *corev1.ConfigMap:
		return t.Data["release"], nil
	}
	return "", ErrNotHelmRelease
}

// decodeHelm3Release decodes the bytes of data into a release
// type. Data must contain a base64 encoded gzipped string of a
// valid release, otherwise an error is returned.
// Almost copy from project helm.
func decodeHelm3Release(data string) (*release.Release, error) {
	// base64 decode string
	b, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}

	// For backwards compatibility with releases that were stored before
	// compression was introduced we skip decompression if the
	// gzip magic header is not found
	if bytes.Equal(b[0:3], magicGzip) {
		r, err := gzip.NewReader(bytes.NewReader(b))
		if err != nil {
			return nil, err
		}
		defer r.Close()
		b2, err := ioutil.ReadAll(r)
		if err != nil {
			return nil, err
		}
		b = b2
	}

	var rls release.Release
	// unmarshal release object bytes
	if err := json.Unmarshal(b, &rls); err != nil {
		return nil, err
	}
	return &rls, nil
}
