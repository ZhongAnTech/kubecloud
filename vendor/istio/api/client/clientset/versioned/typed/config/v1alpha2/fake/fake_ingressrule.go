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

// FakeIngressRules implements IngressRuleInterface
type FakeIngressRules struct {
	Fake *FakeConfigV1alpha2
	ns   string
}

var ingressrulesResource = schema.GroupVersionResource{Group: "config.istio.io", Version: "v1alpha2", Resource: "ingressrules"}

var ingressrulesKind = schema.GroupVersionKind{Group: "config.istio.io", Version: "v1alpha2", Kind: "IngressRule"}

// Get takes name of the ingressRule, and returns the corresponding ingressRule object, and an error if there is any.
func (c *FakeIngressRules) Get(name string, options v1.GetOptions) (result *v1alpha2.IngressRule, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(ingressrulesResource, c.ns, name), &v1alpha2.IngressRule{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha2.IngressRule), err
}

// List takes label and field selectors, and returns the list of IngressRules that match those selectors.
func (c *FakeIngressRules) List(opts v1.ListOptions) (result *v1alpha2.IngressRuleList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(ingressrulesResource, ingressrulesKind, c.ns, opts), &v1alpha2.IngressRuleList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha2.IngressRuleList{}
	for _, item := range obj.(*v1alpha2.IngressRuleList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested ingressRules.
func (c *FakeIngressRules) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(ingressrulesResource, c.ns, opts))

}

// Create takes the representation of a ingressRule and creates it.  Returns the server's representation of the ingressRule, and an error, if there is any.
func (c *FakeIngressRules) Create(ingressRule *v1alpha2.IngressRule) (result *v1alpha2.IngressRule, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(ingressrulesResource, c.ns, ingressRule), &v1alpha2.IngressRule{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha2.IngressRule), err
}

// Update takes the representation of a ingressRule and updates it. Returns the server's representation of the ingressRule, and an error, if there is any.
func (c *FakeIngressRules) Update(ingressRule *v1alpha2.IngressRule) (result *v1alpha2.IngressRule, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(ingressrulesResource, c.ns, ingressRule), &v1alpha2.IngressRule{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha2.IngressRule), err
}

// Delete takes name of the ingressRule and deletes it. Returns an error if one occurs.
func (c *FakeIngressRules) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(ingressrulesResource, c.ns, name), &v1alpha2.IngressRule{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeIngressRules) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(ingressrulesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha2.IngressRuleList{})
	return err
}

// Patch applies the patch and returns the patched ingressRule.
func (c *FakeIngressRules) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha2.IngressRule, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(ingressrulesResource, c.ns, name, data, subresources...), &v1alpha2.IngressRule{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha2.IngressRule), err
}
