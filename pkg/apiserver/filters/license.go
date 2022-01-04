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
	"sync/atomic"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/tools/cache"

	"kubesphere.io/kubesphere/pkg/simple/client/k8s"
	"kubesphere.io/kubesphere/pkg/simple/client/license/utils"

	"k8s.io/klog"

	"kubesphere.io/kubesphere/pkg/apiserver/request"
	"kubesphere.io/kubesphere/pkg/constants"
	licensetypes "kubesphere.io/kubesphere/pkg/simple/client/license/types/v1alpha1"
)

// WithLicense checks whether the license is valid for these clusters.
// If the license is not valid, forbid all the WRITE operations,
// and add HTTP headers to the get requests.
func WithLicense(handler http.Handler, informer cache.SharedInformer, client k8s.Client) http.Handler {
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
	// All the verb comes form pkg/apiserver/request/requestinfo.go#196
	var forbiddenVerb = map[string]bool{
		"create": true,
		"update": true,
		"delete": true,
		"patch":  true,
	}

	go syncLicenseStatus(informer)

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		info, ok := request.RequestInfoFrom(req.Context())
		if !ok {
			klog.Error("Unable to retrieve request info from request")
			handler.ServeHTTP(w, req)
			return
		}

		if vio := getLicenseViolation(); vio.Type != licensetypes.NoViolation {
			// License is invalid.
			verb := info.Verb
			if _, exists := forbiddenVerb[verb]; exists {
				if strings.HasPrefix(info.Path, "/kapis/license.kubesphere.io/") || strings.HasPrefix(info.Path, "/oauth/") ||
					(verb == "delete" && strings.HasPrefix(info.Path, "/kapis/cluster.kubesphere.io/") ||
						// The first login, user needs to change the password.
						(verb == "patch" && strings.HasPrefix(info.Path, "/apis/iam.kubesphere.io/")) ||
						// This request is not a write operation, just transform the Jenkins file to JSON format.
						(verb == "create" && strings.HasPrefix(info.Path, "/kapis/devops.kubesphere.io/v1alpha2/tojson"))) {
					handler.ServeHTTP(w, req)
				} else {
					klog.V(4).Infof("forbidden path: %s, verb: %s, reason: %s", info.Path, info.Verb, vio.Type)
					w.WriteHeader(licensetypes.LicenseViolationCode)
				}
			} else {
				// Return the violation type, so all the GET requests know the status of the license, then the console
				// will show this info to the user.
				w.Header().Add(string(licensetypes.ViolationType), vio.Type)
				if vio.Expected != 0 {
					klog.V(4).Infof("violation type: %s, expected: %d, current: %d", vio.Type, vio.Expected, vio.Current)
					w.Header().Add(string(licensetypes.ViolationExpectedResourceCount), strconv.Itoa(vio.Expected))
					w.Header().Add(string(licensetypes.ViolationCurrentResourceCount), strconv.Itoa(vio.Current))
				}

				if vio.EndTime != nil {
					klog.V(4).Infof("violation type: %s, end time: %v, start-time: %v", vio.Type, vio.EndTime, vio.StartTime)
					w.Header().Add(string(licensetypes.ViolationLicenseEndTime), fmt.Sprintf("%v", vio.EndTime))
					w.Header().Add(string(licensetypes.ViolationLicenseStartTime), fmt.Sprintf("%v", vio.StartTime))
				}
				handler.ServeHTTP(w, req)
			}
		} else {
			handler.ServeHTTP(w, req)
		}
	})
}

var cachedViolation atomic.Value

// syncLicenseStatus sync the license status to cachedViolation.
// Then every request loads the status from the atomic value.
func syncLicenseStatus(informer cache.SharedInformer) {
	informer.AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: func(obj interface{}) bool {
			objMeta, _ := meta.Accessor(obj)
			return objMeta.GetName() == licensetypes.LicenseName && objMeta.GetNamespace() == constants.KubeSphereNamespace
		},
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				objMeta, _ := meta.Accessor(obj)
				stats := objMeta.GetAnnotations()[licensetypes.LicenseStatusKey]
				vio := parseLicenseStatus([]byte(stats))
				cachedViolation.Store(vio)
			},
			DeleteFunc: func(obj interface{}) {
				vio := &licensetypes.Violation{
					Type: licensetypes.EmptyLicense,
				}
				cachedViolation.Store(vio)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				objMeta, _ := meta.Accessor(newObj)
				stats := objMeta.GetAnnotations()[licensetypes.LicenseStatusKey]
				vio := parseLicenseStatus([]byte(stats))
				cachedViolation.Store(vio)
			},
		},
	})
}

func parseLicenseStatus(status []byte) *licensetypes.Violation {
	vio := &licensetypes.Violation{}
	if len(status) > 0 {
		var licenseStatus licensetypes.LicenseStatus
		err := json.Unmarshal(status, &licenseStatus)
		if err != nil {
			vio.Type = licensetypes.FormatError
		} else {
			vio = &licenseStatus.Violation
		}
	} else {
		vio.Type = licensetypes.EmptyLicense
	}
	return vio
}

// getLicenseViolation load the license status from atomic value.
func getLicenseViolation() *licensetypes.Violation {
	if cached := cachedViolation.Load(); cached != nil {
		return cached.(*licensetypes.Violation)
	} else {
		return &licensetypes.Violation{Type: licensetypes.EmptyLicense}
	}
}
