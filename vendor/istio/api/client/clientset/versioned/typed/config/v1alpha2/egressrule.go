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

package v1alpha2

import (
	scheme "istio/api/client/clientset/versioned/scheme"
	v1alpha2 "istio/api/routing/v1alpha2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// EgressRulesGetter has a method to return a EgressRuleInterface.
// A group's client should implement this interface.
type EgressRulesGetter interface {
	EgressRules(namespace string) EgressRuleInterface
}

// EgressRuleInterface has methods to work with EgressRule resources.
type EgressRuleInterface interface {
	Create(*v1alpha2.EgressRule) (*v1alpha2.EgressRule, error)
	Update(*v1alpha2.EgressRule) (*v1alpha2.EgressRule, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha2.EgressRule, error)
	List(opts v1.ListOptions) (*v1alpha2.EgressRuleList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha2.EgressRule, err error)
	EgressRuleExpansion
}

// egressRules implements EgressRuleInterface
type egressRules struct {
	client rest.Interface
	ns     string
}

// newEgressRules returns a EgressRules
func newEgressRules(c *ConfigV1alpha2Client, namespace string) *egressRules {
	return &egressRules{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the egressRule, and returns the corresponding egressRule object, and an error if there is any.
func (c *egressRules) Get(name string, options v1.GetOptions) (result *v1alpha2.EgressRule, err error) {
	result = &v1alpha2.EgressRule{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("egressrules").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of EgressRules that match those selectors.
func (c *egressRules) List(opts v1.ListOptions) (result *v1alpha2.EgressRuleList, err error) {
	result = &v1alpha2.EgressRuleList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("egressrules").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested egressRules.
func (c *egressRules) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("egressrules").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a egressRule and creates it.  Returns the server's representation of the egressRule, and an error, if there is any.
func (c *egressRules) Create(egressRule *v1alpha2.EgressRule) (result *v1alpha2.EgressRule, err error) {
	result = &v1alpha2.EgressRule{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("egressrules").
		Body(egressRule).
		Do().
		Into(result)
	return
}

// Update takes the representation of a egressRule and updates it. Returns the server's representation of the egressRule, and an error, if there is any.
func (c *egressRules) Update(egressRule *v1alpha2.EgressRule) (result *v1alpha2.EgressRule, err error) {
	result = &v1alpha2.EgressRule{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("egressrules").
		Name(egressRule.Name).
		Body(egressRule).
		Do().
		Into(result)
	return
}

// Delete takes name of the egressRule and deletes it. Returns an error if one occurs.
func (c *egressRules) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("egressrules").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *egressRules) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("egressrules").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched egressRule.
func (c *egressRules) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha2.EgressRule, err error) {
	result = &v1alpha2.EgressRule{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("egressrules").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
