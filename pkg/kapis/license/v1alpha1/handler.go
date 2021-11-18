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
	"encoding/json"
	"errors"

	restful "github.com/emicklei/go-restful"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog"

	"kubesphere.io/kubesphere/pkg/simple/client/license/clusterinfo"

	"kubesphere.io/kubesphere/pkg/informers"
	servererr "kubesphere.io/kubesphere/pkg/server/errors"
	"kubesphere.io/kubesphere/pkg/simple/client/license/client"
	"kubesphere.io/kubesphere/pkg/simple/client/multicluster"

	"kubesphere.io/kubesphere/pkg/api"
	"kubesphere.io/kubesphere/pkg/constants"
	"kubesphere.io/kubesphere/pkg/simple/client/license/cert"
	licensetypes "kubesphere.io/kubesphere/pkg/simple/client/license/types/v1alpha1"
)

type licensesInterface interface {
	GetLicense(req *restful.Request, resp *restful.Response)
	UpdateLicense(req *restful.Request, resp *restful.Response)
	DeleteLicense(req *restful.Request, resp *restful.Response)
}

type licenseHandler struct {
	client clientset.Interface

	nodeLister v1.NodeLister

	cim          *clusterinfo.ClusterInfoManager
	multiCluster bool
}

func newLicenseHandler(client clientset.Interface, informerFactory informers.InformerFactory, opts *multicluster.Options) licensesInterface {
	handler := licenseHandler{
		client: client,
	}

	if opts != nil && opts.Enable {
		handler.cim = clusterinfo.NewClusterInfoManager(informerFactory.KubeSphereSharedInformerFactory().Cluster().V1alpha1().Clusters(), nil)
		handler.multiCluster = true
		go handler.cim.Run(wait.NeverStop)
	} else {
		handler.nodeLister = informerFactory.KubernetesSharedInformerFactory().Core().V1().Nodes().Lister()
	}

	return &handler
}

// UpdateLicense update current license by user input.
func (h *licenseHandler) UpdateLicense(req *restful.Request, resp *restful.Response) {
	licenseResp := &client.License{
		Status: &licensetypes.LicenseStatus{
			Violation: licensetypes.Violation{
				Type: licensetypes.NoViolation,
			},
		},
	}
	err := req.ReadEntity(licenseResp)
	if err != nil || licenseResp.Data == nil {
		api.HandleBadRequest(resp, nil, err)
		return
	}

	vio, err := licenseResp.Data.Check(cert.CertStore.Cert, "")
	if err != nil {
		klog.Errorf("check license failed, error: %s", err)
		api.HandleError(resp, nil, err)
		return
	}

	if vio != nil {
		licenseResp.Status.Violation = *vio
		klog.V(2).Infof("check license failed, violation type: %s, reason: %s", vio.Type, vio.Reason)
		resp.WriteAsJson(licenseResp)
		return
	}

	// update license
	err = licenseResp.Data.SaveLicenseData(h.client.CoreV1().Secrets(constants.KubeSphereNamespace))
	if err != nil {
		klog.Errorf("update license failed, error: %s", err)
		api.HandleInternalError(resp, nil, errors.New("update license failed"))
		return
	} else {
		klog.V(2).Infof("license updated")
		resp.WriteAsJson(licenseResp)
	}
}

func (h *licenseHandler) GetLicense(req *restful.Request, resp *restful.Response) {
	license := &client.License{
		Status: &licensetypes.LicenseStatus{
			Violation: licensetypes.Violation{
				Type: licensetypes.NoViolation,
			},
		},
	}
	ctx := context.Background()
	secret, err := h.client.CoreV1().Secrets(constants.KubeSphereNamespace).
		Get(ctx, licensetypes.LicenseName, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			klog.Errorf("get license failed, error: %s", err)
			api.HandleError(resp, nil, err)
		}
	}

	// Build the data filed.
	var licenseData = &licensetypes.License{}

	if len(secret.Data[licensetypes.LicenseKey]) > 0 {
		licenseData, err = licensetypes.LoadLicense(secret.Data[licensetypes.LicenseKey])
		if err != nil {
			klog.Errorf("parse license data failed, error: %s", err)
			license.Status.Violation.Type = licensetypes.FormatError
			resp.WriteAsJson(license)
			return
		}
		license.Data = licenseData
	} else {
		license.Status.Violation.Type = licensetypes.EmptyLicense
	}

	// Build the status field.
	// The status of the license is saved into the annotations of the secret.
	if data := secret.Annotations[licensetypes.LicenseStatusKey]; len(data) > 0 {
		// fast path, get cluster status from secret's annotations
		err := json.Unmarshal([]byte(secret.Annotations[licensetypes.LicenseStatusKey]), &license.Status)
		if err != nil {
			license.Status.Violation.Type = licensetypes.FormatError
			klog.Errorf("get license status failed, error: %s", err)
			api.HandleError(resp, nil, err)
			return
		}
	} else {
		// slow path, collect cluster status from api.
		status, err := h.collectClusterInfo(ctx)
		if err != nil {
			klog.Errorf("get license status failed, error: %s", err)
			api.HandleError(resp, nil, err)
			return
		}
		status.Violation = license.Status.Violation
		license.Status = &status
	}

	resp.WriteAsJson(license)
}

// collectClusterInfo collect the cpu num, cpu core and node num of the host cluster and member cluster.
func (h *licenseHandler) collectClusterInfo(ctx context.Context) (licenseStatus licensetypes.LicenseStatus, err error) {
	var cn []clusterinfo.ClusterNode

	if h.multiCluster {
		// Get nodes from the host cluster and member clusters.
		cn, licenseStatus.ClusterNum, err = h.cim.GetClusterInfo()
		if err != nil {
			return
		}
	} else {
		// Get nodes from the current cluster.
		var nodes *corev1.NodeList
		nodes, err = h.client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		if err != nil {
			return
		}

		cn = make([]clusterinfo.ClusterNode, len(nodes.Items))
		for i := range nodes.Items {
			cn[i] = clusterinfo.ClusterNode{Node: &nodes.Items[i]}
		}
		licenseStatus.ClusterNum = 1
	}

	for _, node := range cn {
		coreNum := 0
		coreCapacity := node.Node.Status.Capacity.Cpu()
		if coreCapacity != nil {
			num, _ := coreCapacity.AsInt64()
			coreNum += int(num)
		} else {
			klog.V(4).Infof("cpu core is empty for %s/%s", node.Cluster, node.Node.Name)
		}

		// The cluster name has been set to empty, to differentiate it from a node that comes from a member cluster.
		if node.Cluster == "" {
			licenseStatus.Host.CoreNum += coreNum
			licenseStatus.Host.NodeNum += 1
		} else {
			licenseStatus.Member.CoreNum += coreNum
			licenseStatus.Member.NodeNum += 1
		}
	}

	return
}

// DeleteLicense delete license secret
func (h *licenseHandler) DeleteLicense(req *restful.Request, resp *restful.Response) {
	err := h.client.CoreV1().Secrets(constants.KubeSphereNamespace).Delete(context.Background(), licensetypes.LicenseName, metav1.DeleteOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return
		}
		klog.Errorf("delete license failed, error: %s", err)
		api.HandleError(resp, nil, err)
		return
	}

	klog.Infof("license deleted")
	resp.WriteEntity(servererr.None)
}
