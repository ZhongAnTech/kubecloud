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

// This file was automatically generated by informer-gen

package v1alpha2

import (
	versioned "istio/api/client/clientset/versioned"
	internalinterfaces "istio/api/client/informers/externalversions/internalinterfaces"
	v1alpha2 "istio/api/client/listers/config/v1alpha2"
	routing_v1alpha2 "istio/api/routing/v1alpha2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
	time "time"
)

// EgressRuleInformer provides access to a shared informer and lister for
// EgressRules.
type EgressRuleInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha2.EgressRuleLister
}

type egressRuleInformer struct {
	factory internalinterfaces.SharedInformerFactory
}

// NewEgressRuleInformer constructs a new informer for EgressRule type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewEgressRuleInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				return client.ConfigV1alpha2().EgressRules(namespace).List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				return client.ConfigV1alpha2().EgressRules(namespace).Watch(options)
			},
		},
		&routing_v1alpha2.EgressRule{},
		resyncPeriod,
		indexers,
	)
}

func defaultEgressRuleInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewEgressRuleInformer(client, v1.NamespaceAll, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
}

func (f *egressRuleInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&routing_v1alpha2.EgressRule{}, defaultEgressRuleInformer)
}

func (f *egressRuleInformer) Lister() v1alpha2.EgressRuleLister {
	return v1alpha2.NewEgressRuleLister(f.Informer().GetIndexer())
}
