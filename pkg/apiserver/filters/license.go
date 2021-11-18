//go:build !license_disable
// +build !license_disable

/*
Copyright 2021 KubeSphere Authors

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

package filters

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"kubesphere.io/kubesphere/pkg/simple/client/k8s"
	"kubesphere.io/kubesphere/pkg/simple/client/license/utils"

	v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog"

	"kubesphere.io/kubesphere/pkg/apiserver/request"
	"kubesphere.io/kubesphere/pkg/constants"
	licensetypes "kubesphere.io/kubesphere/pkg/simple/client/license/types/v1alpha1"
)

// WithLicense checks whether the license is valid for these clusters.
// If the license is not valid, forbid all the WRITE operations,
// and add HTTP headers to the get requests.
func WithLicense(handler http.Handler, lister v1.SecretLister, client k8s.Client) http.Handler {
	role, err := utils.ClusterRole(context.Background(), client.Config())
	if err != nil {
		klog.Errorf("get cluster role failed, error: %s", err)
		return handler
	}

	// The member cluster need not run this filter.
	if role == "member" {
		klog.V(4).Infof("current cluster is member cluster, skip license check")
		return handler
	}

	// If the license is invalid, forbid all the WRITE operations.
	var forbiddenVerb = map[string]bool{
		"post":   true,
		"put":    true,
		"delete": true,
		"patch":  true,
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		info, ok := request.RequestInfoFrom(req.Context())
		if !ok {
			klog.Error("Unable to retrieve request info from request")
			handler.ServeHTTP(w, req)
			return
		}

		if vio := getLicenseViolation(lister); vio.Type != licensetypes.NoViolation {
			// License is invalid.
			if _, exists := forbiddenVerb[info.Verb]; exists {
				if strings.HasPrefix(info.Path, "/kapis/license.v1") || strings.HasPrefix(info.Path, "/oauth/") ||
					(info.Verb == "delete" && strings.HasPrefix(info.Path, "/kapis/cluster")) {
					handler.ServeHTTP(w, req)
				} else {
					klog.V(2).Infof("forbidden path: %s, verb: %s, reason: %s", info.Path, info.Verb, vio.Type)
					w.WriteHeader(licensetypes.LicenseViolationCode)
				}
			} else {
				// Return the violation type, so all the GET requests know the status of the license, then the console
				// will show this info to the user.
				w.Header().Add("violation-type", vio.Type)
				if vio.Expected != 0 {
					klog.V(4).Infof("violation type: %s, expected: %d, current: %d", vio.Type, vio.Expected, vio.Current)
					w.Header().Add("violation-expected", strconv.Itoa(vio.Expected))
					w.Header().Add("violation-current", strconv.Itoa(vio.Current))
				}

				if vio.EndTime != nil {
					klog.V(4).Infof("violation type: %s, end time: %v, start-time: %v", vio.Type, vio.EndTime, vio.StartTime)
					w.Header().Add("violation-end-time", fmt.Sprintf("%v", vio.EndTime))
					w.Header().Add("violation-start-time", fmt.Sprintf("%v", vio.StartTime))
				}
				handler.ServeHTTP(w, req)
			}
		} else {
			handler.ServeHTTP(w, req)
		}
	})
}

func getLicenseViolation(lister v1.SecretLister) *licensetypes.Violation {
	secret, err := lister.Secrets(constants.KubeSphereNamespace).Get(licensetypes.LicenseName)

	if err != nil {
		klog.Errorf("get license failed, error: %s", err)
		return &licensetypes.Violation{
			Type: licensetypes.EmptyLicense,
		}
	}

	sts := secret.Annotations[licensetypes.LicenseStatusKey]
	if len(sts) > 0 {
		var licenseStatus licensetypes.LicenseStatus
		err = json.Unmarshal([]byte(sts), &licenseStatus)
		if err != nil {
			return &licensetypes.Violation{
				Type: licensetypes.FormatError,
			}
		}
		return &licenseStatus.Violation
	} else {
		return &licensetypes.Violation{
			Type: licensetypes.FormatError,
		}
	}
}
