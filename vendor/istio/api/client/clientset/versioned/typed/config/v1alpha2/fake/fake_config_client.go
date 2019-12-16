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
	v1alpha2 "istio/api/client/clientset/versioned/typed/config/v1alpha2"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeConfigV1alpha2 struct {
	*testing.Fake
}

func (c *FakeConfigV1alpha2) DestinationPolicies(namespace string) v1alpha2.DestinationPolicyInterface {
	return &FakeDestinationPolicies{c, namespace}
}

func (c *FakeConfigV1alpha2) EgressRules(namespace string) v1alpha2.EgressRuleInterface {
	return &FakeEgressRules{c, namespace}
}

func (c *FakeConfigV1alpha2) IngressRules(namespace string) v1alpha2.IngressRuleInterface {
	return &FakeIngressRules{c, namespace}
}

func (c *FakeConfigV1alpha2) RouteRules(namespace string) v1alpha2.RouteRuleInterface {
	return &FakeRouteRules{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeConfigV1alpha2) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
