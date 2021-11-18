// +build client_check_license

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

// client should build with tag `-tags "client_check_license"`, then init() will check whether license is valid or not.
package client

import (
	"context"
	"os"
	"time"

	"k8s.io/klog"
)

func init() {
	client := NewLicenseClient(DefaultKSApiserver)
	klog.V(3).Infof("license check start")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.GetLicense(ctx)
	if err != nil {
		klog.Errorf("check license failed, error: %s", err)
		os.Exit(1)
	} else {
		if resp.Violation.Type != types.NoViolation {
			klog.Errorf("license violation, type: %s, reason: %s", resp.Violation.Type, resp.Violation.Reason)
			os.Exit(1)
		}
	}

	klog.V(3).Infof("license check end")
}
