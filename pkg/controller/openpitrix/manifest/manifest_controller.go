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
	"encoding/base64"
	"fmt"
	"math/rand"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1alpha1 "kubesphere.io/api/cluster/v1alpha1"

	"k8s.io/apimachinery/pkg/api/errors"
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

// To implement backup function, database operator require an existed s3 secret or configmap in the target namespace
// it is difficult to pre sync s3 secret/configmap to every namespace, so we need to do some compatible stuff to
// make sure secret/configmap be created in need. just works when the special placeholder was set in customResource.
const (
	managedNamespace    = "dmp-system"
	managedS3SecretName = "dmp-managed-s3-secret"

	s3SecretPlaceHolder = "$DMP-MANAGED-S3-SECRET$"

	PgS3FieldMarker          = "$DMP-PG-S3-"
	PgS3KeyPlaceHolder       = PgS3FieldMarker + "KEY$"
	PgS3KeySecretPlaceHolder = PgS3FieldMarker + "KEY-SECRET$"
	PgS3BucketPlaceHolder    = PgS3FieldMarker + "BUCKET$"
	PgS3EndpointPlaceHolder  = PgS3FieldMarker + "ENDPOINT$"
	PgS3RegionPlaceHolder    = PgS3FieldMarker + "REGION$"
	PgS3URIStylePlaceHolder  = PgS3FieldMarker + "URI-STYLE$"
)

var (
	decUnstructured = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	finalizer       = "openpitrix.kubesphere.io/manifest"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func (r *ManifestReconciler) getNextReconcileTime(manifest *v1alpha1.Manifest) time.Duration {
	now := time.Now()

	// If status->version was updated in the last 10 seconds,
	// status of custom resource likely changed in these 10 seconds.
	// thus we should reduce the waiting time.
	if !manifest.Status.LastUpdate.IsZero() &&
		manifest.Status.LastUpdate.After(now.Add(-10*time.Second)) {
		return 2 * time.Second
	}

	// a random value in range [5,30) returned
	return time.Duration(rand.Intn(30-5)+5) * time.Second
}

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

	// get cluster client with kubeconfig
	cli, err := r.getClusterClient(manifest.GetManifestCluster())
	if err != nil {
		return ctrl.Result{}, err
	}

	if !manifest.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.deleteManifest(ctx, cli, manifest)
	}

	if manifest.Status.State == "" {
		return ctrl.Result{}, r.updateManifestState(ctx, manifest, v1alpha1.StatusCreating, "")
	}

	if manifest.Status.State == v1alpha1.StatusCreating {
		if !sliceutil.HasString(manifest.ObjectMeta.Finalizers, finalizer) {
			return r.setManifestOwnerAndFinalizer(ctx, manifest)
		}
		return r.installManifest(ctx, cli, manifest)
	}

	// check weather the custom resource still exists
	if !r.IsCustomResourceExisted(ctx, cli, manifest) {
		if manifest.Status.ResourceState != v1alpha1.StatusDeleted {
			return ctrl.Result{}, r.updateManifestState(ctx, manifest, "", v1alpha1.StatusDeleted)
		}
		return ctrl.Result{RequeueAfter: r.getNextReconcileTime(manifest)}, nil
	}

	// reconcile customResource to desired state by compare manifest->spec->version with manifest->status->version
	if manifest.Status.Version != manifest.Spec.Version {
		return r.updateManifest(ctx, cli, manifest)
	}

	// sync customResource->status->state to manifest->status->resourceState periodically
	return r.checkResourceStatus(ctx, cli, manifest)
}

func (r *ManifestReconciler) setManifestOwnerAndFinalizer(ctx context.Context, manifest *v1alpha1.Manifest) (ctrl.Result, error) {
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

func (r *ManifestReconciler) updateManifest(ctx context.Context, cli client.Client, manifest *v1alpha1.Manifest) (ctrl.Result, error) {
	// get unstructured object
	objSlice, err := getUnstructuredObj(manifest)
	if err != nil || len(objSlice) == 0 {
		return ctrl.Result{}, err
	}

	// update the first resource, the first resource is manifest
	newObj := objSlice[0].DeepCopy()
	err = cli.Get(ctx, types.NamespacedName{
		Namespace: objSlice[0].GetNamespace(),
		Name:      objSlice[0].GetName(),
	}, newObj)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	newObj.Object["spec"] = objSlice[0].Object["spec"]
	err = cli.Update(ctx, newObj)
	if err != nil {
		klog.Errorf("update custom resource error: %s", err)
		return ctrl.Result{}, err
	}

	manifest.Status.Version = manifest.Spec.Version
	manifest.Status.LastUpdate = &metav1.Time{Time: time.Now()}
	err = r.Status().Update(ctx, manifest)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ManifestReconciler) deleteManifest(ctx context.Context, cli client.Client, manifest *v1alpha1.Manifest) (ctrl.Result, error) {
	if !sliceutil.HasString(manifest.ObjectMeta.Finalizers, finalizer) {
		return ctrl.Result{}, nil
	}

	objSlice, err := getUnstructuredObj(manifest)
	if err != nil {
		klog.Errorf("get unstructured object error: %s", err.Error())
	}

	// delete all resources in manifest
	for i := range objSlice {
		err = cli.Delete(ctx, objSlice[i])
		if err != nil {
			klog.Errorf("delete custom resource error: %s, %s/%s", err, objSlice[i].GetNamespace(), objSlice[i].GetName())
			continue
		}
	}

	// remove backup secret or configmap
	if err = r.deleteManagedResource(ctx, cli, manifest); err != nil {
		klog.Errorf("error:%s occurred while deleting managed resource", err)
	}

	// remove finalizer
	manifest.ObjectMeta.Finalizers = sliceutil.RemoveString(manifest.ObjectMeta.Finalizers, func(item string) bool {
		if item == finalizer {
			return true
		}
		return false
	})

	return ctrl.Result{}, r.Update(ctx, manifest)
}

func (r *ManifestReconciler) installManifest(ctx context.Context, cli client.Client, manifest *v1alpha1.Manifest) (ctrl.Result, error) {
	// replace managed fields
	if r.hasManagedField(manifest) {
		return r.replaceManagedFields(ctx, manifest)
	}

	// create backup secret or configmap
	if r.hasManagedResource(manifest) {
		return r.createManagedResource(ctx, manifest)
	}

	objSlice, err := getUnstructuredObj(manifest)
	if err != nil {
		return ctrl.Result{}, err
	}

	// create all resources in manifest
	if len(objSlice) == 0 {
		klog.Errorf("manifest without custom resource is invalid")
		return ctrl.Result{}, r.updateManifestState(ctx, manifest, v1alpha1.StatusError, v1alpha1.StatusFailed)
	}
	// create the related-resources first, and then create the custom-resource.
	for i := len(objSlice) - 1; i >= 0; i-- {
		err = cli.Get(ctx, types.NamespacedName{
			Namespace: objSlice[i].GetNamespace(),
			Name:      objSlice[i].GetName(),
		}, objSlice[i])
		if err != nil && !errors.IsNotFound(err) {
			return ctrl.Result{}, r.updateManifestState(ctx, manifest, v1alpha1.StatusFailed, v1alpha1.StatusFailed)
		}

		err = cli.Create(ctx, objSlice[i])
		if err != nil && !errors.IsAlreadyExists(err) {
			klog.Errorf("create custom resource error: %s, %s/%s", err, objSlice[i].GetNamespace(), objSlice[i].GetName())
			return ctrl.Result{}, r.updateManifestState(ctx, manifest, v1alpha1.StatusError, v1alpha1.StatusFailed)
		}
	}

	// update manifest state and version after created manifest
	manifest.Status.Version = manifest.Spec.Version
	return ctrl.Result{}, r.updateManifestState(ctx, manifest, v1alpha1.StatusCreated, v1alpha1.StatusCreating)
}

func (r *ManifestReconciler) hasManagedField(manifest *v1alpha1.Manifest) bool {
	if strings.Contains(manifest.Spec.CustomResource, PgS3FieldMarker) {
		return true
	}
	return false
}

func (r *ManifestReconciler) hasManagedResource(manifest *v1alpha1.Manifest) bool {
	if strings.Contains(manifest.Spec.CustomResource, s3SecretPlaceHolder) {
		return true
	}
	return false
}

// replaceManagedFields replace holders with fixed value base on dmp-managed-s3-secret in dmp-system namespace
func (r *ManifestReconciler) replaceManagedFields(ctx context.Context, manifest *v1alpha1.Manifest) (ctrl.Result, error) {
	secret := &corev1.Secret{}
	err := r.Client.Get(ctx, types.NamespacedName{Namespace: managedNamespace, Name: managedS3SecretName}, secret)
	if err != nil {
		klog.Errorf("%s not found in %s, unable to create custom resource that contains %s", managedS3SecretName, managedNamespace, s3SecretPlaceHolder)
		return ctrl.Result{}, err
	}

	placeholders := []string{PgS3KeyPlaceHolder, PgS3KeySecretPlaceHolder, PgS3BucketPlaceHolder, PgS3EndpointPlaceHolder, PgS3RegionPlaceHolder, PgS3URIStylePlaceHolder}
	for _, holder := range placeholders {
		key := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(holder, PgS3FieldMarker, "pg-s3-"), `$`, ""))
		value := secret.Data[key]
		valueString := string(value)

		if holder == PgS3KeyPlaceHolder || holder == PgS3KeySecretPlaceHolder {
			valueString = base64.StdEncoding.EncodeToString(value)
		}

		manifest.Spec.CustomResource = strings.ReplaceAll(manifest.Spec.CustomResource, holder, valueString)
	}

	return ctrl.Result{}, r.Update(ctx, manifest)
}

// createManagedResource create s3 secret/configmap base on fixed secret/configmap in dmp-system namespace
func (r *ManifestReconciler) createManagedResource(ctx context.Context, manifest *v1alpha1.Manifest) (ctrl.Result, error) {
	cli, err := r.getClusterClient(manifest.GetManifestCluster())
	if err != nil {
		return ctrl.Result{}, err
	}

	if strings.Contains(manifest.Spec.CustomResource, s3SecretPlaceHolder) {
		secret := &corev1.Secret{}
		err = r.Client.Get(ctx, types.NamespacedName{Namespace: managedNamespace, Name: managedS3SecretName}, secret)
		if err != nil {
			klog.Errorf("%s not found in %s, unable to create custom resource that contains %s", managedS3SecretName, managedNamespace, s3SecretPlaceHolder)
			return ctrl.Result{}, err
		}
		derivedSecret := &corev1.Secret{}
		derivedSecret.Namespace = manifest.Spec.Namespace
		derivedSecret.Name = fmt.Sprintf("%s-%s", manifest.Name, managedS3SecretName)
		derivedSecret.Data = secret.Data

		err = cli.Create(ctx, derivedSecret)
		if err != nil && !errors.IsAlreadyExists(err) {
			return ctrl.Result{}, err
		}

		manifest.Spec.CustomResource = strings.ReplaceAll(manifest.Spec.CustomResource, s3SecretPlaceHolder, derivedSecret.Name)
		return ctrl.Result{}, r.Update(ctx, manifest)
	}

	return ctrl.Result{}, nil
}

// createManagedResource clean up s3 secret/configmap when manifest is deleted
func (r *ManifestReconciler) deleteManagedResource(ctx context.Context, cli client.Client, manifest *v1alpha1.Manifest) error {
	if strings.Contains(manifest.Spec.CustomResource, managedS3SecretName) {
		secret := &corev1.Secret{}
		secret.Namespace = manifest.Spec.Namespace
		secret.Name = fmt.Sprintf("%s-%s", manifest.Name, managedS3SecretName)

		err := cli.Delete(ctx, secret)
		return client.IgnoreNotFound(err)
	}

	return nil
}

func (r *ManifestReconciler) IsCustomResourceExisted(ctx context.Context, cli client.Client, manifest *v1alpha1.Manifest) bool {
	objSlice, err := getUnstructuredObj(manifest)
	if err != nil || len(objSlice) == 0 {
		return true
	}
	cli, err = r.getClusterClient(manifest.GetManifestCluster())
	if err != nil {
		return true
	}

	err = cli.Get(ctx, types.NamespacedName{
		Namespace: manifest.Spec.Namespace,
		Name:      manifest.Name}, objSlice[0])
	if err != nil && errors.IsNotFound(err) {
		klog.Warningf("custom resource for %s not found, maybe manually deleted by administrator", manifest.Name)
		return false
	}

	return true
}

func (r *ManifestReconciler) updateManifestState(ctx context.Context, manifest *v1alpha1.Manifest, state, resourceState string) error {
	if state != "" {
		manifest.Status.State = state
	}
	if resourceState != "" {
		manifest.Status.ResourceState = resourceState
	}

	return r.Status().Update(ctx, manifest)
}

func (r *ManifestReconciler) checkResourceStatus(ctx context.Context, cli client.Client, manifest *v1alpha1.Manifest) (ctrl.Result, error) {
	// periodically check the status of the custom resource
	objSlice, err := getUnstructuredObj(manifest)
	if err != nil || len(objSlice) == 0 {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	err = cli.Get(ctx, types.NamespacedName{
		Namespace: manifest.Spec.Namespace,
		Name:      manifest.Name}, objSlice[0])
	if err != nil {
		klog.V(1).Info(err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// get custom resource state
	err = r.Status().Update(ctx, updateUnstructuredObjStatus(manifest, objSlice))
	if err != nil {
		klog.Errorf("update manifest status error: %s", err)
	}
	return ctrl.Result{RequeueAfter: r.getNextReconcileTime(manifest)}, err
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
			}
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
