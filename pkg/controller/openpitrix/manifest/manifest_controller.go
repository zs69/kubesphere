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

package manifest

import (
	"context"
	"time"

	clusterv1alpha1 "kubesphere.io/api/cluster/v1alpha1"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"kubesphere.io/api/application/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"kubesphere.io/kubesphere/pkg/client/informers/externalversions"
	"kubesphere.io/kubesphere/pkg/utils/clusterclient"
	"kubesphere.io/kubesphere/pkg/utils/sliceutil"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const CheckTime = 30 * time.Second

var (
	decUnstructured = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	finalizer       = "openpitrix.kubesphere.io/manifest"
)

// ManifestReconciler reconciles a Manifest object
type ManifestReconciler struct {
	client.Client
	KsFactory          externalversions.SharedInformerFactory
	MultiClusterEnable bool
	clusterClients     clusterclient.ClusterClients
	Scheme             *runtime.Scheme
}

func (r *ManifestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	manifest := &v1alpha1.Manifest{}
	if err := r.Get(ctx, req.NamespacedName, manifest); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if manifest.Status.State == "" {
		manifest.Status.State = v1alpha1.StatusCreating
		err := r.Status().Update(ctx, manifest)
		return reconcile.Result{}, err
	}

	if manifest.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted
		if !sliceutil.HasString(manifest.ObjectMeta.Finalizers, finalizer) {
			clusterName := manifest.GetManifestCluster()
			if r.MultiClusterEnable && clusterName != "" {
				clusterInfo, err := r.clusterClients.Get(clusterName)
				if err != nil {
					// cluster not exists, delete the manifest
					klog.Warningf("cluster %s not found, delete the custom resource %s/%s",
						clusterName, manifest.GetManifestNamespace(), manifest.GetName())
					return reconcile.Result{}, r.Delete(context.TODO(), manifest)
				}

				// Host cluster will self-healing, delete host cluster won't cause deletion of manifest
				if !r.clusterClients.IsHostCluster(clusterInfo) {
					// add owner References
					manifest.OwnerReferences = append(manifest.OwnerReferences, metav1.OwnerReference{
						APIVersion: clusterv1alpha1.SchemeGroupVersion.String(),
						Kind:       clusterv1alpha1.ResourceKindCluster,
						Name:       clusterInfo.Name,
						UID:        clusterInfo.UID,
					})
				}
			}
			manifest.ObjectMeta.Finalizers = append(manifest.ObjectMeta.Finalizers, finalizer)
			err := r.Update(ctx, manifest)
			return reconcile.Result{}, err
		}
	} else {
		// The object is being deleted
		if sliceutil.HasString(manifest.ObjectMeta.Finalizers, finalizer) {
			err := r.deleteManifest(ctx, manifest)
			if err != nil {
				klog.Errorf("delete custom resource error: %s", client.IgnoreNotFound(err).Error())
			}
			manifest.ObjectMeta.Finalizers = sliceutil.RemoveString(manifest.ObjectMeta.Finalizers, func(item string) bool {
				if item == finalizer {
					return true
				}
				return false
			})
			err = r.Update(ctx, manifest)
			return reconcile.Result{}, err
		}
	}
	// if manifest state is creating, to install custom resource
	if manifest.Status.State == v1alpha1.StatusCreating {
		if err := r.installManifest(ctx, manifest); err != nil {
			return ctrl.Result{}, err
		}
	} else if manifest.Status.Version != manifest.Spec.Version {
		return r.updateManifest(ctx, manifest)
	} else {
		// check custom resources status
		return r.checkResourceStatus(ctx, manifest)
	}
	return ctrl.Result{}, nil
}

func (r *ManifestReconciler) updateManifest(ctx context.Context, manifest *v1alpha1.Manifest) (ctrl.Result, error) {
	// get cluster client with kubeconfig
	cli, err := r.getClusterClient(manifest.GetManifestCluster())
	if err != nil {
		return ctrl.Result{}, err
	}

	// get unstructured object
	objSlice, err := getUnstructuredObj(manifest)
	if err != nil || len(objSlice) == 0 {
		return ctrl.Result{}, err
	}

	// patch the first resource, the first resource is manifest
	oldObj := objSlice[0].DeepCopy()
	err = cli.Get(ctx, types.NamespacedName{
		Namespace: objSlice[0].GetNamespace(),
		Name:      objSlice[0].GetName(),
	}, oldObj)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	oldObj.Object["spec"] = objSlice[0].Object["spec"]
	err = cli.Patch(ctx, oldObj, client.Merge)
	if err != nil {
		klog.Errorf("update custom resource error: %s", err)
		return ctrl.Result{}, err
	}

	manifest.Status.Version = manifest.Spec.Version
	err = r.Status().Update(ctx, manifest)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ManifestReconciler) deleteManifest(ctx context.Context, manifest *v1alpha1.Manifest) error {
	// get member cluster client
	cli, err := r.getClusterClient(manifest.GetManifestCluster())
	if err != nil {
		return err
	}

	objSlice, err := getUnstructuredObj(manifest)
	if err != nil {
		klog.Errorf("get unstructured object error: %s", err.Error())
		return err
	}
	// delete all resources in manifest
	for i := range objSlice {
		err = cli.Delete(ctx, objSlice[i])
		if err != nil {
			klog.Errorf("delete custom resource error: %s, %s/%s", err, objSlice[i].GetNamespace(), objSlice[i].GetName())
			continue
		}
	}
	return client.IgnoreNotFound(err)
}

func (r *ManifestReconciler) installManifest(ctx context.Context, manifest *v1alpha1.Manifest) error {
	// install custom resource in host or member cluster
	cli, err := r.getClusterClient(manifest.GetManifestCluster())
	if err != nil {
		return err
	}

	objSlice, err := getUnstructuredObj(manifest)
	if err != nil {
		return err
	}

	// create all resources in manifest
	for i := range objSlice {
		err = cli.Get(ctx, types.NamespacedName{
			Namespace: objSlice[i].GetNamespace(),
			Name:      objSlice[i].GetName(),
		}, objSlice[i])
		if err != nil && errors.IsAlreadyExists(err) {
			return r.updateManifestState(manifest, v1alpha1.StatusFailed, v1alpha1.StatusFailed)
		}

		err = cli.Create(ctx, objSlice[i])
		if err != nil {
			klog.Errorf("create custom resource error: %s, %s/%s", err, objSlice[i].GetNamespace(), objSlice[i].GetName())
			return r.updateManifestState(manifest, v1alpha1.StatusError, v1alpha1.StatusFailed)
		}
	}
	// update manifest state and version after created manifest
	manifest.Status.Version = manifest.Spec.Version
	return r.updateManifestState(manifest, v1alpha1.StatusCreated, v1alpha1.StatusCreating)
}

func (r *ManifestReconciler) updateManifestState(manifest *v1alpha1.Manifest, state, resourceState string) error {
	manifest.Status.State = state
	manifest.Status.ResourceState = resourceState
	err := r.Status().Update(context.TODO(), manifest)
	return err
}

func (r *ManifestReconciler) checkResourceStatus(ctx context.Context, manifest *v1alpha1.Manifest) (ctrl.Result, error) {
	// periodically check the status of the custom resource
	objSlice, err := getUnstructuredObj(manifest)
	if err != nil || len(objSlice) == 0 {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	cli, err := r.getClusterClient(manifest.GetManifestCluster())
	if err != nil {
		return ctrl.Result{}, err
	}
	err = cli.Get(ctx, types.NamespacedName{
		Namespace: manifest.Spec.Namespace,
		Name:      manifest.Name}, objSlice[0])
	if err != nil {
		klog.V(1).Info(err.Error())
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// get custom resource state
	err = r.Status().Update(ctx, updateUnstructuredObjStatus(manifest, objSlice))
	if err != nil {
		klog.Errorf("update manifest status error: %s", err)
	}
	return ctrl.Result{RequeueAfter: CheckTime}, err
}

func getUnstructuredObj(manifest *v1alpha1.Manifest) (objSlice []*unstructured.Unstructured, err error) {
	obj := &unstructured.Unstructured{}
	_, _, err = decUnstructured.Decode([]byte(manifest.Spec.CustomResource), nil, obj)
	if err != nil {
		klog.Errorf("decode unstructured object error: %s", err.Error())
	}
	objSlice = append(objSlice, obj)

	// If there are related resources, they need to be created together with custom resource
	if manifest.Spec.RelatedResources != nil {
		for i := range manifest.Spec.RelatedResources {
			resource := &unstructured.Unstructured{}
			_, _, err = decUnstructured.Decode([]byte(manifest.Spec.RelatedResources[i].Data), nil, resource)
			if err != nil {
				klog.Errorf("decode unstructured object error: %s", err.Error())
				continue
			}
			objSlice = append(objSlice, resource)
		}
	}
	return
}

func updateUnstructuredObjStatus(manifest *v1alpha1.Manifest, obj []*unstructured.Unstructured) *v1alpha1.Manifest {
	for i := range obj {
		statusMap, ok := obj[i].Object["status"].(map[string]interface{})
		if ok {
			resourceState, ok := statusMap["state"].(string)
			if ok {
				if i == 0 {
					// update custom resource state
					manifest.Status.ResourceState = resourceState
				} else {
					// update related resource state
					relatedRes := &v1alpha1.RelatedResourceState{
						ResourceName:  obj[i].GetName(),
						ResourceState: resourceState,
					}
					manifest.Status.RelatedResourceStates = append(manifest.Status.RelatedResourceStates, relatedRes)
				}
			} else {
				continue
			}
		} else {
			continue
		}
	}
	return manifest
}

// SetupWithManager sets up the controller with the Manager.
func (r *ManifestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.Client == nil {
		r.Client = mgr.GetClient()
	}
	if r.Scheme == nil {
		r.Scheme = mgr.GetScheme()
	}
	if r.KsFactory != nil && r.MultiClusterEnable {
		r.clusterClients = clusterclient.NewClusterClient(r.KsFactory.Cluster().V1alpha1().Clusters())
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Manifest{}).
		Complete(r)
}
