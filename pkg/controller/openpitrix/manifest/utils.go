package manifest

import (
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ManifestReconciler) getClusterClient(clusterName string) (client.Client, error) {
	var clusterCli client.Client
	if r.MultiClusterEnable && clusterName != "" {
		clusterInfo, err := r.clusterClients.Get(clusterName)
		if err != nil {
			klog.Errorf("get cluster(%s) info error: %s", clusterName, err)
			return nil, err
		}

		kubeconfig, err := clientcmd.RESTConfigFromKubeConfig(clusterInfo.Spec.Connection.KubeConfig)
		if err != nil {
			klog.Errorf("get cluster config error: %s", err)
			return nil, err
		}
		clusterCli, err = client.New(kubeconfig, client.Options{Scheme: r.Scheme})
		if err != nil {
			klog.Errorf("get cluster client with kubeconfig error: %s", err)
			return nil, err
		}
	} else {
		return r.Client, nil
	}
	return clusterCli, nil
}
