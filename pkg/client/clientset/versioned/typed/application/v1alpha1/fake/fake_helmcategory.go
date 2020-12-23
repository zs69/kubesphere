/*
Copyright 2020 The KubeSphere Authors.

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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
	v1alpha1 "kubesphere.io/kubesphere/pkg/apis/application/v1alpha1"
)

// FakeHelmCategories implements HelmCategoryInterface
type FakeHelmCategories struct {
	Fake *FakeApplicationV1alpha1
}

var helmcategoriesResource = schema.GroupVersionResource{Group: "application.kubesphere.io", Version: "v1alpha1", Resource: "helmcategories"}

var helmcategoriesKind = schema.GroupVersionKind{Group: "application.kubesphere.io", Version: "v1alpha1", Kind: "HelmCategory"}

// Get takes name of the helmCategory, and returns the corresponding helmCategory object, and an error if there is any.
func (c *FakeHelmCategories) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.HelmCategory, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(helmcategoriesResource, name), &v1alpha1.HelmCategory{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.HelmCategory), err
}

// List takes label and field selectors, and returns the list of HelmCategories that match those selectors.
func (c *FakeHelmCategories) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.HelmCategoryList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(helmcategoriesResource, helmcategoriesKind, opts), &v1alpha1.HelmCategoryList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.HelmCategoryList{ListMeta: obj.(*v1alpha1.HelmCategoryList).ListMeta}
	for _, item := range obj.(*v1alpha1.HelmCategoryList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested helmCategories.
func (c *FakeHelmCategories) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(helmcategoriesResource, opts))
}

// Create takes the representation of a helmCategory and creates it.  Returns the server's representation of the helmCategory, and an error, if there is any.
func (c *FakeHelmCategories) Create(ctx context.Context, helmCategory *v1alpha1.HelmCategory, opts v1.CreateOptions) (result *v1alpha1.HelmCategory, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(helmcategoriesResource, helmCategory), &v1alpha1.HelmCategory{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.HelmCategory), err
}

// Update takes the representation of a helmCategory and updates it. Returns the server's representation of the helmCategory, and an error, if there is any.
func (c *FakeHelmCategories) Update(ctx context.Context, helmCategory *v1alpha1.HelmCategory, opts v1.UpdateOptions) (result *v1alpha1.HelmCategory, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(helmcategoriesResource, helmCategory), &v1alpha1.HelmCategory{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.HelmCategory), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeHelmCategories) UpdateStatus(ctx context.Context, helmCategory *v1alpha1.HelmCategory, opts v1.UpdateOptions) (*v1alpha1.HelmCategory, error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateSubresourceAction(helmcategoriesResource, "status", helmCategory), &v1alpha1.HelmCategory{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.HelmCategory), err
}

// Delete takes name of the helmCategory and deletes it. Returns an error if one occurs.
func (c *FakeHelmCategories) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(helmcategoriesResource, name), &v1alpha1.HelmCategory{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeHelmCategories) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(helmcategoriesResource, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.HelmCategoryList{})
	return err
}

// Patch applies the patch and returns the patched helmCategory.
func (c *FakeHelmCategories) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.HelmCategory, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(helmcategoriesResource, name, pt, data, subresources...), &v1alpha1.HelmCategory{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.HelmCategory), err
}
