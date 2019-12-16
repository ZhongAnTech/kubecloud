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

// FakeEgressRules implements EgressRuleInterface
type FakeEgressRules struct {
	Fake *FakeConfigV1alpha2
	ns   string
}

var egressrulesResource = schema.GroupVersionResource{Group: "config.istio.io", Version: "v1alpha2", Resource: "egressrules"}

var egressrulesKind = schema.GroupVersionKind{Group: "config.istio.io", Version: "v1alpha2", Kind: "EgressRule"}

// Get takes name of the egressRule, and returns the corresponding egressRule object, and an error if there is any.
func (c *FakeEgressRules) Get(name string, options v1.GetOptions) (result *v1alpha2.EgressRule, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(egressrulesResource, c.ns, name), &v1alpha2.EgressRule{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha2.EgressRule), err
}

// List takes label and field selectors, and returns the list of EgressRules that match those selectors.
func (c *FakeEgressRules) List(opts v1.ListOptions) (result *v1alpha2.EgressRuleList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(egressrulesResource, egressrulesKind, c.ns, opts), &v1alpha2.EgressRuleList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha2.EgressRuleList{}
	for _, item := range obj.(*v1alpha2.EgressRuleList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested egressRules.
func (c *FakeEgressRules) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(egressrulesResource, c.ns, opts))

}

// Create takes the representation of a egressRule and creates it.  Returns the server's representation of the egressRule, and an error, if there is any.
func (c *FakeEgressRules) Create(egressRule *v1alpha2.EgressRule) (result *v1alpha2.EgressRule, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(egressrulesResource, c.ns, egressRule), &v1alpha2.EgressRule{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha2.EgressRule), err
}

// Update takes the representation of a egressRule and updates it. Returns the server's representation of the egressRule, and an error, if there is any.
func (c *FakeEgressRules) Update(egressRule *v1alpha2.EgressRule) (result *v1alpha2.EgressRule, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(egressrulesResource, c.ns, egressRule), &v1alpha2.EgressRule{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha2.EgressRule), err
}

// Delete takes name of the egressRule and deletes it. Returns an error if one occurs.
func (c *FakeEgressRules) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(egressrulesResource, c.ns, name), &v1alpha2.EgressRule{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeEgressRules) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(egressrulesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha2.EgressRuleList{})
	return err
}

// Patch applies the patch and returns the patched egressRule.
func (c *FakeEgressRules) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha2.EgressRule, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(egressrulesResource, c.ns, name, data, subresources...), &v1alpha2.EgressRule{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha2.EgressRule), err
}
