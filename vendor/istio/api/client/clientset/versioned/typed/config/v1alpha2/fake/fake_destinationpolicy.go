/*
Copyright 2017 The Kubernetes Authors.

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

package fake

import (
	v1alpha2 "istio/api/routing/v1alpha2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeDestinationPolicies implements DestinationPolicyInterface
type FakeDestinationPolicies struct {
	Fake *FakeConfigV1alpha2
	ns   string
}

var destinationpoliciesResource = schema.GroupVersionResource{Group: "config.istio.io", Version: "v1alpha2", Resource: "destinationpolicies"}

var destinationpoliciesKind = schema.GroupVersionKind{Group: "config.istio.io", Version: "v1alpha2", Kind: "DestinationPolicy"}

// Get takes name of the destinationPolicy, and returns the corresponding destinationPolicy object, and an error if there is any.
func (c *FakeDestinationPolicies) Get(name string, options v1.GetOptions) (result *v1alpha2.DestinationPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(destinationpoliciesResource, c.ns, name), &v1alpha2.DestinationPolicy{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha2.DestinationPolicy), err
}

// List takes label and field selectors, and returns the list of DestinationPolicies that match those selectors.
func (c *FakeDestinationPolicies) List(opts v1.ListOptions) (result *v1alpha2.DestinationPolicyList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(destinationpoliciesResource, destinationpoliciesKind, c.ns, opts), &v1alpha2.DestinationPolicyList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha2.DestinationPolicyList{}
	for _, item := range obj.(*v1alpha2.DestinationPolicyList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested destinationPolicies.
func (c *FakeDestinationPolicies) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(destinationpoliciesResource, c.ns, opts))

}

// Create takes the representation of a destinationPolicy and creates it.  Returns the server's representation of the destinationPolicy, and an error, if there is any.
func (c *FakeDestinationPolicies) Create(destinationPolicy *v1alpha2.DestinationPolicy) (result *v1alpha2.DestinationPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(destinationpoliciesResource, c.ns, destinationPolicy), &v1alpha2.DestinationPolicy{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha2.DestinationPolicy), err
}

// Update takes the representation of a destinationPolicy and updates it. Returns the server's representation of the destinationPolicy, and an error, if there is any.
func (c *FakeDestinationPolicies) Update(destinationPolicy *v1alpha2.DestinationPolicy) (result *v1alpha2.DestinationPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(destinationpoliciesResource, c.ns, destinationPolicy), &v1alpha2.DestinationPolicy{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha2.DestinationPolicy), err
}

// Delete takes name of the destinationPolicy and deletes it. Returns an error if one occurs.
func (c *FakeDestinationPolicies) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(destinationpoliciesResource, c.ns, name), &v1alpha2.DestinationPolicy{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeDestinationPolicies) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(destinationpoliciesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha2.DestinationPolicyList{})
	return err
}

// Patch applies the patch and returns the patched destinationPolicy.
func (c *FakeDestinationPolicies) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha2.DestinationPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(destinationpoliciesResource, c.ns, name, data, subresources...), &v1alpha2.DestinationPolicy{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha2.DestinationPolicy), err
}
