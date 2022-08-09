package clusterlabel

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	clusterv1alpha1 "kubesphere.io/api/cluster/v1alpha1"
)

// LabelReconciler is a reconciler for the Label object.
type LabelReconciler struct {
	client.Client
}

// Reconcile reconciles the Label object, sync label to the individual Cluster CRs.
func (r *LabelReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	label := &clusterv1alpha1.Label{}
	if err := r.Get(ctx, req.NamespacedName, label); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if label.DeletionTimestamp != nil {
		return ctrl.Result{}, r.deleteLabel(ctx, label)
	}

	if len(label.Finalizers) == 0 {
		label.Finalizers = []string{clusterv1alpha1.LabelFinalizer}
		if err := r.Update(ctx, label); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{}, r.syncLabelToClusters(ctx, label)
}

func (r *LabelReconciler) syncLabelToClusters(ctx context.Context, label *clusterv1alpha1.Label) error {
	klog.V(4).Infof("sync label %s[%s/%v] to clusters: %v", label.Name, label.Spec.Key, label.Spec.Value, label.Spec.Clusters)
	clusterSets := sets.NewString(label.Spec.Clusters...)
	for name := range clusterSets {
		cluster := &clusterv1alpha1.Cluster{}
		if err := r.Get(ctx, client.ObjectKey{Name: name}, cluster); err != nil {
			if errors.IsNotFound(err) {
				clusterSets.Delete(name)
				continue
			} else {
				return err
			}
		}

		if cluster.Labels == nil {
			cluster.Labels = make(map[string]string)
		}
		if _, ok := cluster.Labels[fmt.Sprintf(clusterv1alpha1.ClusterLabelFormat, label.Name)]; ok {
			continue
		}
		cluster.Labels[fmt.Sprintf(clusterv1alpha1.ClusterLabelFormat, label.Name)] = ""
		if err := r.Update(ctx, cluster); err != nil {
			return err
		}
	}
	clusters := clusterSets.List()
	// some clusters have been deleted and this list needs to be updated
	if len(clusters) != len(label.Spec.Clusters) {
		label.Spec.Clusters = clusters
		return r.Update(ctx, label)
	}
	return nil
}

func (r *LabelReconciler) deleteLabel(ctx context.Context, label *clusterv1alpha1.Label) error {
	klog.V(4).Infof("deleting label %s, removing cluster %v related label", label.Name, label.Spec.Clusters)
	for _, name := range label.Spec.Clusters {
		cluster := &clusterv1alpha1.Cluster{}
		if err := r.Get(ctx, client.ObjectKey{Name: name}, cluster); err != nil {
			if errors.IsNotFound(err) {
				continue
			} else {
				return err
			}
		}
		delete(cluster.Labels, fmt.Sprintf(clusterv1alpha1.ClusterLabelFormat, label.Name))
		if err := r.Update(ctx, cluster); err != nil {
			return err
		}
	}
	label.Finalizers = nil
	return r.Update(ctx, label)
}

// InjectClient is used to inject the client into LabelReconciler.
func (r *LabelReconciler) InjectClient(c client.Client) error {
	r.Client = c
	return nil
}

// SetupWithManager setups the LabelReconciler with manager.
func (r *LabelReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return builder.
		ControllerManagedBy(mgr).
		For(
			&clusterv1alpha1.Label{},
			builder.WithPredicates(
				predicate.ResourceVersionChangedPredicate{},
			),
		).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 2,
		}).
		Complete(r)
}

// ClusterReconciler is a reconciler for the Cluster object.
type ClusterReconciler struct {
	client.Client
}

// Reconcile reconciles the Cluster object, sync annotaions' label IDs to the individual Label CRs.
func (r *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	cluster := &clusterv1alpha1.Cluster{}
	if err := r.Get(ctx, req.NamespacedName, cluster); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	annotations := cluster.Annotations
	if len(annotations) == 0 {
		return ctrl.Result{}, nil
	}
	labelString := annotations[clusterv1alpha1.ClusterLabelIDsAnnotation]
	if labelString == "" {
		return ctrl.Result{}, nil
	}

	labels := strings.Split(labelString, ",")
	klog.V(4).Infof("sync cluster %s to labels: %v", cluster.Name, labels)
	for _, name := range labels {
		label := &clusterv1alpha1.Label{}
		if err := r.Get(ctx, client.ObjectKey{Name: strings.TrimSpace(name)}, label); err != nil {
			if errors.IsNotFound(err) {
				continue
			} else {
				return ctrl.Result{}, err
			}
		}
		clusters := sets.NewString(label.Spec.Clusters...)
		if clusters.Has(cluster.Name) {
			continue
		}
		clusters.Insert(cluster.Name)
		label.Spec.Clusters = clusters.List()
		if err := r.Update(ctx, label); err != nil {
			return ctrl.Result{}, err
		}
	}

	delete(annotations, clusterv1alpha1.ClusterLabelIDsAnnotation)
	cluster.Annotations = annotations
	return ctrl.Result{}, r.Update(ctx, cluster)
}

// InjectClient is used to inject the client into ClusterReconciler.
func (r *ClusterReconciler) InjectClient(c client.Client) error {
	r.Client = c
	return nil
}

// SetupWithManager setups the ClusterReconciler with manager.
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return builder.
		ControllerManagedBy(mgr).
		For(
			&clusterv1alpha1.Cluster{},
			builder.WithPredicates(
				predicate.ResourceVersionChangedPredicate{},
			),
		).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 2,
		}).
		Complete(r)
}
