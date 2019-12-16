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

// IngressRulesGetter has a method to return a IngressRuleInterface.
// A group's client should implement this interface.
type IngressRulesGetter interface {
	IngressRules(namespace string) IngressRuleInterface
}

// IngressRuleInterface has methods to work with IngressRule resources.
type IngressRuleInterface interface {
	Create(*v1alpha2.IngressRule) (*v1alpha2.IngressRule, error)
	Update(*v1alpha2.IngressRule) (*v1alpha2.IngressRule, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha2.IngressRule, error)
	List(opts v1.ListOptions) (*v1alpha2.IngressRuleList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha2.IngressRule, err error)
	IngressRuleExpansion
}

// ingressRules implements IngressRuleInterface
type ingressRules struct {
	client rest.Interface
	ns     string
}

// newIngressRules returns a IngressRules
func newIngressRules(c *ConfigV1alpha2Client, namespace string) *ingressRules {
	return &ingressRules{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the ingressRule, and returns the corresponding ingressRule object, and an error if there is any.
func (c *ingressRules) Get(name string, options v1.GetOptions) (result *v1alpha2.IngressRule, err error) {
	result = &v1alpha2.IngressRule{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("ingressrules").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of IngressRules that match those selectors.
func (c *ingressRules) List(opts v1.ListOptions) (result *v1alpha2.IngressRuleList, err error) {
	result = &v1alpha2.IngressRuleList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("ingressrules").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested ingressRules.
func (c *ingressRules) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("ingressrules").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a ingressRule and creates it.  Returns the server's representation of the ingressRule, and an error, if there is any.
func (c *ingressRules) Create(ingressRule *v1alpha2.IngressRule) (result *v1alpha2.IngressRule, err error) {
	result = &v1alpha2.IngressRule{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("ingressrules").
		Body(ingressRule).
		Do().
		Into(result)
	return
}

// Update takes the representation of a ingressRule and updates it. Returns the server's representation of the ingressRule, and an error, if there is any.
func (c *ingressRules) Update(ingressRule *v1alpha2.IngressRule) (result *v1alpha2.IngressRule, err error) {
	result = &v1alpha2.IngressRule{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("ingressrules").
		Name(ingressRule.Name).
		Body(ingressRule).
		Do().
		Into(result)
	return
}

// Delete takes name of the ingressRule and deletes it. Returns an error if one occurs.
func (c *ingressRules) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("ingressrules").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *ingressRules) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("ingressrules").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched ingressRule.
func (c *ingressRules) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha2.IngressRule, err error) {
	result = &v1alpha2.IngressRule{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("ingressrules").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
