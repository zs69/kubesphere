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

package license

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"sync"
	"time"

	k8sinformers "k8s.io/client-go/informers"

	ksinformers "kubesphere.io/kubesphere/pkg/client/informers/externalversions"

	"kubesphere.io/kubesphere/pkg/simple/client/license/utils"

	"k8s.io/apimachinery/pkg/types"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"

	"kubesphere.io/kubesphere/pkg/simple/client/license/clusterinfo"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"kubesphere.io/kubesphere/pkg/constants"
	"kubesphere.io/kubesphere/pkg/simple/client/license/cert"
	licensetype "kubesphere.io/kubesphere/pkg/simple/client/license/types/v1alpha1"
)

type LicenseController struct {
	client.Client

	nodeInformer v1.NodeInformer

	cert *x509.Certificate

	stopCh <-chan struct{}
	lock   sync.Mutex
	// multi cluster mode is enabled or not
	multiCluster bool
	restConfig   *rest.Config

	eventChan chan *clusterinfo.ClusterNodeEvent
	cim       *clusterinfo.ClusterInfoManager

	k8s k8sinformers.SharedInformerFactory
	ks  ksinformers.SharedInformerFactory
}

// NewLicenseController create a controller the watch the nodes info and license info.
// 1. The controller will fetch all the node info from member clusters and the host cluster if the cluster is a host cluster.
// 2. If the cluster is just a cluster which multi-cluster mode is not enable, the controller just watch the node of this cluster.
// 3. If the cluster is a member cluster, this controller will just exit.
func NewLicenseController(config *rest.Config, k8s k8sinformers.SharedInformerFactory, ks ksinformers.SharedInformerFactory, multiCluster bool, stopCh <-chan struct{}) *LicenseController {
	err := cert.InitCert()
	if err != nil {
		klog.Errorf("init cert failed, error: %s", err)
	}
	return &LicenseController{
		cert:         cert.CertStore.Cert,
		stopCh:       stopCh,
		multiCluster: multiCluster,
		restConfig:   config,
		k8s:          k8s,
		ks:           ks,
	}
}

func (lc *LicenseController) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	// TODO replace the ks-license secret with CRD.
	// We just care about the ks-license secret.
	if request.Namespace != constants.KubeSphereNamespace || request.Name != licensetype.LicenseName {
		return reconcile.Result{}, nil
	}

	klog.V(4).Infof("license has changed, sync license start")
	err := lc.syncLicenseStatus(ctx)
	klog.V(4).Infof("sync license end")
	return reconcile.Result{}, err
}

// collectClusterInfo calculate the cor num, node num and cluster num from all the clusters.
// Then the license controller will check whether the license is valid or not.
func (lc *LicenseController) collectClusterInfo(ctx context.Context) (licenseStatus licensetype.LicenseStatus, err error) {
	var cn []clusterinfo.ClusterNode

	if lc.multiCluster {
		// Get nodes from the host cluster and member clusters.
		cn, licenseStatus.ClusterNum, err = lc.cim.GetClusterInfo()
		if err != nil {
			return
		}
	} else {
		// Get nodes from the current cluster.
		var nodes corev1.NodeList
		err = lc.Client.List(ctx, &nodes)
		if err != nil {
			return
		}

		cn = make([]clusterinfo.ClusterNode, len(nodes.Items))
		for i := range nodes.Items {
			cn[i] = clusterinfo.ClusterNode{Node: &nodes.Items[i]}
		}
		licenseStatus.ClusterNum = 1
	}

	// Start to calculate core num and node num.
	for _, node := range cn {
		coreNum := 0
		coreCapacity := node.Node.Status.Capacity.Cpu()
		if coreCapacity != nil {
			num, _ := coreCapacity.AsInt64()
			coreNum += int(num)
		} else {
			klog.V(4).Infof("cpu core is empty for %s/%s", node.Cluster, node.Node.Name)
		}

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

func checkLicense(clusterStats *licensetype.LicenseStatus, secret *corev1.Secret) (violation *licensetype.Violation, err error) {
	var license *licensetype.License
	if len(secret.Data[licensetype.LicenseKey]) == 0 {
		violation = &licensetype.Violation{Type: licensetype.EmptyLicense}
		return
	}

	license, err = licensetype.LoadLicense(secret.Data[licensetype.LicenseKey])
	if err != nil {
		violation.Type = licensetype.FormatError
	} else {
		violation, err = license.Check(cert.CertStore.Cert, "")
	}

	if violation == nil {
		violation = &licensetype.Violation{Type: licensetype.NoViolation}
		switch license.LicenseType {
		// subscription mode, checks the cluster num and node num.
		case licensetype.LicenseTypeSubscription:
			if clusterStats.ClusterNum > license.MaxCluster {
				return &licensetype.Violation{
					Type:     licensetype.ClusterCountLimitExceeded,
					Current:  clusterStats.ClusterNum,
					Expected: license.MaxCluster,
				}, nil
			}
			if clusterStats.Host.NodeNum+clusterStats.Member.NodeNum > license.MaxNode {
				return &licensetype.Violation{
					Type:     licensetype.NodeCountLimitExceeded,
					Current:  clusterStats.Host.NodeNum + clusterStats.Member.NodeNum,
					Expected: license.MaxNode,
				}, nil
			}
		// maintenance mode, just checks the core num.
		case licensetype.LicenseTypeMaintenance:
			if clusterStats.Host.CoreNum+clusterStats.Member.CoreNum > license.MaxCore {
				return &licensetype.Violation{
					Type:     licensetype.CoreCountLimitExceeded,
					Expected: license.MaxCore,
					Current:  clusterStats.Host.CoreNum + clusterStats.Member.CoreNum,
				}, nil
			}
		// managed mode, checks the core num on the host cluster and cluster num.
		case licensetype.LicenseTypeManged:
			if clusterStats.ClusterNum > license.MaxCluster {
				return &licensetype.Violation{
					Type:     licensetype.ClusterCountLimitExceeded,
					Current:  clusterStats.ClusterNum,
					Expected: license.MaxCluster,
				}, nil
			}
			if clusterStats.Host.CoreNum > license.MaxCore {
				return &licensetype.Violation{
					Type:     licensetype.CoreCountLimitExceeded,
					Expected: license.MaxCore,
					Current:  clusterStats.Host.CoreNum,
				}, nil
			}

		default:
			klog.V(4).Infof("invalid license type: %s", license.LicenseType)
			violation.Type = licensetype.InvalidLicenseType
		}
	}

	return
}

// syncLicenseStatus check whether the license is valid or not and save the status of cluster and license to annotation.
func (lc *LicenseController) syncLicenseStatus(ctx context.Context) error {
	secret := &corev1.Secret{}
	err := lc.Client.Get(ctx,
		types.NamespacedName{Namespace: constants.KubeSphereNamespace, Name: licensetype.LicenseName}, secret)

	if apierrors.IsNotFound(err) {
		klog.Errorf("license not found")
		return nil
	}

	klog.V(4).Infof("collect cluster info")
	cs, err := lc.collectClusterInfo(ctx)
	if err == nil {
		klog.V(4).Infof("check the license whether is valid or not")
		vio, err := checkLicense(&cs, secret)
		if err != nil {
			klog.Errorf("check license error: %s", err)
		}
		cs.Violation = *vio
	} else {
		klog.Errorf("collect cluster info failed, error: %s", err)
	}

	newSecret := secret.DeepCopy()
	if newSecret.Annotations == nil {
		newSecret.Annotations = map[string]string{}
	}

	// save license status to annotations, so the ks-apiserver could get the status of the license.
	statusStr, _ := json.Marshal(cs)
	newSecret.Annotations[licensetype.LicenseStatusKey] = string(statusStr)

	return lc.patchSecret(ctx, secret, newSecret)
}

// patchSecret patches the ks-license secret
func (lc *LicenseController) patchSecret(ctx context.Context, old, new *corev1.Secret) error {
	patch := client.MergeFrom(old)
	data, _ := patch.Data(new)

	// data = "{}"
	if len(data) == 2 {
		klog.V(4).Infof("there is no update for secret %s", old.Name)
		return nil
	} else {
		klog.V(4).Infof("start to patch secret %s", new.Name)
	}

	err := lc.Client.Patch(ctx, new, client.MergeFrom(old), &client.PatchOptions{})

	if err != nil {
		if apierrors.IsNotFound(err) {
			klog.V(4).Infof("patch license failed, license not found")
			return nil
		} else {
			klog.Errorf("patch license failed, error: %s", err)
			return err
		}
	}

	return nil
}

func (lc *LicenseController) SetupWithManager(mgr ctrl.Manager) error {
	lc.Client = mgr.GetClient()

	role, err := utils.ClusterRole(context.Background(), lc.restConfig)
	if err != nil {
		return err
	}

	if role == "host" {
		// If the cluster is a host cluster, the controller must fetch all the node's info from host cluster and member clusters.
		lc.eventChan = make(chan *clusterinfo.ClusterNodeEvent, 200)
		lc.cim = clusterinfo.NewClusterInfoManager(lc.ks.Cluster().V1alpha1().Clusters(), lc.eventChan)
		go func() {
			<-mgr.Elected()
			go lc.cim.Run(lc.stopCh)
			ticker := time.NewTicker(5 * time.Minute)
			for {
				select {
				case <-lc.eventChan:
					klog.V(4).Infof("node has changed, sync license start")
					lc.syncLicenseStatus(context.Background())
					klog.V(4).Infof("sync license end")
				case <-ticker.C:
					klog.V(4).Infof("periodically sync license started")
					lc.syncLicenseStatus(context.Background())
					klog.V(4).Infof("sync license end")
				}
			}
		}()
	} else if role == "" {
		// Multi cluster mode is not enabled. The controller just watch the nodes of the cluster.
		lc.nodeInformer = lc.k8s.Core().V1().Nodes()
		i := lc.nodeInformer.Informer()
		i.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				klog.V(4).Infof("node has changed, sync license start")
				lc.syncLicenseStatus(context.Background())
				klog.V(4).Infof("sync license start")
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				klog.V(4).Infof("node has changed, sync license start")
				lc.syncLicenseStatus(context.Background())
				klog.V(4).Infof("sync license start")
			},
			DeleteFunc: func(obj interface{}) {
				klog.V(4).Infof("node has changed, sync license start")
				lc.syncLicenseStatus(context.Background())
				klog.V(4).Infof("sync license start")
			},
		})
		go func() {
			<-mgr.Elected()
			i.Run(lc.stopCh)
		}()
	} else {
		// Member cluster
		// Not run this cluster on member cluster.
		return nil
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Secret{}).Complete(lc)
}
