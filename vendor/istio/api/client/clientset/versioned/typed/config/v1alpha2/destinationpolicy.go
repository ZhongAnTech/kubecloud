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

// DestinationPoliciesGetter has a method to return a DestinationPolicyInterface.
// A group's client should implement this interface.
type DestinationPoliciesGetter interface {
	DestinationPolicies(namespace string) DestinationPolicyInterface
}

// DestinationPolicyInterface has methods to work with DestinationPolicy resources.
type DestinationPolicyInterface interface {
	Create(*v1alpha2.DestinationPolicy) (*v1alpha2.DestinationPolicy, error)
	Update(*v1alpha2.DestinationPolicy) (*v1alpha2.DestinationPolicy, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha2.DestinationPolicy, error)
	List(opts v1.ListOptions) (*v1alpha2.DestinationPolicyList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha2.DestinationPolicy, err error)
	DestinationPolicyExpansion
}

// destinationPolicies implements DestinationPolicyInterface
type destinationPolicies struct {
	client rest.Interface
	ns     string
}

// newDestinationPolicies returns a DestinationPolicies
func newDestinationPolicies(c *ConfigV1alpha2Client, namespace string) *destinationPolicies {
	return &destinationPolicies{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the destinationPolicy, and returns the corresponding destinationPolicy object, and an error if there is any.
func (c *destinationPolicies) Get(name string, options v1.GetOptions) (result *v1alpha2.DestinationPolicy, err error) {
	result = &v1alpha2.DestinationPolicy{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("destinationpolicies").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of DestinationPolicies that match those selectors.
func (c *destinationPolicies) List(opts v1.ListOptions) (result *v1alpha2.DestinationPolicyList, err error) {
	result = &v1alpha2.DestinationPolicyList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("destinationpolicies").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested destinationPolicies.
func (c *destinationPolicies) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("destinationpolicies").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a destinationPolicy and creates it.  Returns the server's representation of the destinationPolicy, and an error, if there is any.
func (c *destinationPolicies) Create(destinationPolicy *v1alpha2.DestinationPolicy) (result *v1alpha2.DestinationPolicy, err error) {
	result = &v1alpha2.DestinationPolicy{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("destinationpolicies").
		Body(destinationPolicy).
		Do().
		Into(result)
	return
}

// Update takes the representation of a destinationPolicy and updates it. Returns the server's representation of the destinationPolicy, and an error, if there is any.
func (c *destinationPolicies) Update(destinationPolicy *v1alpha2.DestinationPolicy) (result *v1alpha2.DestinationPolicy, err error) {
	result = &v1alpha2.DestinationPolicy{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("destinationpolicies").
		Name(destinationPolicy.Name).
		Body(destinationPolicy).
		Do().
		Into(result)
	return
}

// Delete takes name of the destinationPolicy and deletes it. Returns an error if one occurs.
func (c *destinationPolicies) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("destinationpolicies").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *destinationPolicies) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("destinationpolicies").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched destinationPolicy.
func (c *destinationPolicies) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha2.DestinationPolicy, err error) {
	result = &v1alpha2.DestinationPolicy{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("destinationpolicies").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
