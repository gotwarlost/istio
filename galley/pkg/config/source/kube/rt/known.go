// Copyright 2019 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rt

import (
	"fmt"
	"reflect"

	"github.com/gogo/protobuf/proto"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	v1beta12 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"istio.io/istio/galley/pkg/config/scope"
	"istio.io/istio/galley/pkg/config/source/kube/apiserver/stats"
)

func (p *Provider) initKnownAdapters() {
	runtimeScheme := runtime.NewScheme()
	codecs := serializer.NewCodecFactory(runtimeScheme)
	deserializer := codecs.UniversalDeserializer()

	p.known = map[string]*Adapter{
		asTypesKey("", "Service"): {
			extractObject: defaultExtractObject,
			extractResource: func(o interface{}) (proto.Message, error) {
				if obj, ok := o.(*v1.Service); ok {
					return &obj.Spec, nil
				}
				return nil, fmt.Errorf("unable to convert to v1.Service: %T", o)
			},
			newInformer: func() (cache.SharedIndexInformer, error) {
				informer, err := p.sharedInformerFactory()
				if err != nil {
					return nil, err
				}

				return informer.Core().V1().Services().Informer(), nil
			},
			parseJSON: func(input []byte) (interface{}, error) {
				out := &v1.Service{}
				if _, _, err := deserializer.Decode(input, nil, out); err != nil {
					return nil, err
				}
				return out, nil
			},
			getStatus: noStatus,
			isEqual:   resourceVersionsMatch,
			isBuiltIn: true,
		},

		asTypesKey("", "Namespace"): {
			extractObject: defaultExtractObject,
			extractResource: func(o interface{}) (proto.Message, error) {
				if obj, ok := o.(*v1.Namespace); ok {
					return &obj.Spec, nil
				}
				return nil, fmt.Errorf("unable to convert to v1.Namespace: %T", o)
			},
			newInformer: func() (cache.SharedIndexInformer, error) {
				informer, err := p.sharedInformerFactory()
				if err != nil {
					return nil, err
				}

				return informer.Core().V1().Namespaces().Informer(), nil
			},
			parseJSON: func(input []byte) (interface{}, error) {
				out := &v1.Namespace{}
				if _, _, err := deserializer.Decode(input, nil, out); err != nil {
					return nil, err
				}
				return out, nil
			},
			getStatus: noStatus,
			isEqual:   resourceVersionsMatch,
			isBuiltIn: true,
		},

		asTypesKey("", "Node"): {
			extractObject: defaultExtractObject,
			extractResource: func(o interface{}) (proto.Message, error) {
				if obj, ok := o.(*v1.Node); ok {
					return &obj.Spec, nil
				}
				return nil, fmt.Errorf("unable to convert to v1.Node: %T", o)
			},
			newInformer: func() (cache.SharedIndexInformer, error) {
				informer, err := p.sharedInformerFactory()
				if err != nil {
					return nil, err
				}

				return informer.Core().V1().Nodes().Informer(), nil
			},
			parseJSON: func(input []byte) (interface{}, error) {
				out := &v1.Node{}
				if _, _, err := deserializer.Decode(input, nil, out); err != nil {
					return nil, err
				}
				return out, nil
			},
			getStatus: noStatus,
			isEqual:   resourceVersionsMatch,
			isBuiltIn: true,
		},

		asTypesKey("", "Pod"): {
			extractObject: defaultExtractObject,
			extractResource: func(o interface{}) (proto.Message, error) {
				if obj, ok := o.(*v1.Pod); ok {
					return obj, nil
				}
				return nil, fmt.Errorf("unable to convert to v1.Pod: %T", o)
			},
			newInformer: func() (cache.SharedIndexInformer, error) {
				informer, err := p.sharedInformerFactory()
				if err != nil {
					return nil, err
				}

				return informer.Core().V1().Pods().Informer(), nil
			},
			parseJSON: func(input []byte) (interface{}, error) {
				out := &v1.Pod{}
				if _, _, err := deserializer.Decode(input, nil, out); err != nil {
					return nil, err
				}
				return out, nil
			},
			getStatus: noStatus,
			isEqual:   resourceVersionsMatch,
			isBuiltIn: true,
		},

		asTypesKey("", "Endpoints"): {
			extractObject: defaultExtractObject,
			extractResource: func(o interface{}) (proto.Message, error) {
				// TODO(nmittler): This copies ObjectMeta since Endpoints have no spec.
				if obj, ok := o.(*v1.Endpoints); ok {
					return obj, nil
				}
				return nil, fmt.Errorf("unable to convert to v1.Endpoints: %T", o)
			},
			newInformer: func() (cache.SharedIndexInformer, error) {
				informer, err := p.sharedInformerFactory()
				if err != nil {
					return nil, err
				}

				return informer.Core().V1().Endpoints().Informer(), nil
			},
			parseJSON: func(input []byte) (interface{}, error) {
				out := &v1.Endpoints{}
				if _, _, err := deserializer.Decode(input, nil, out); err != nil {
					return nil, err
				}
				return out, nil
			},
			isEqual: func(o1 interface{}, o2 interface{}) bool {
				r1, ok1 := o1.(*v1.Endpoints)
				r2, ok2 := o2.(*v1.Endpoints)
				if !ok1 || !ok2 {
					msg := fmt.Sprintf("error decoding kube endpoints during update, o1 type: %T, o2 type: %T",
						o1, o2)
					scope.Source.Error(msg)
					stats.RecordEventError(msg)
					return false
				}
				// Endpoint updates can be noisy. Make sure that the subsets have actually changed.
				return reflect.DeepEqual(r1.Subsets, r2.Subsets)
			},
			getStatus: noStatus,
			isBuiltIn: true,
		},
		asTypesKey("extensions", "Ingress"): {
			extractObject: defaultExtractObject,
			extractResource: func(o interface{}) (proto.Message, error) {
				if obj, ok := o.(*v1beta1.Ingress); ok {
					return &obj.Spec, nil
				}
				return nil, fmt.Errorf("unable to convert to v1beta1.Ingress: %T", o)
			},
			newInformer: func() (cache.SharedIndexInformer, error) {
				informer, err := p.sharedInformerFactory()
				if err != nil {
					return nil, err
				}

				return informer.Extensions().V1beta1().Ingresses().Informer(), nil
			},
			parseJSON: func(input []byte) (interface{}, error) {
				out := &v1beta1.Ingress{}
				if _, _, err := deserializer.Decode(input, nil, out); err != nil {
					return nil, err
				}
				return out, nil
			},
			getStatus: noStatus,
			isEqual:   resourceVersionsMatch,
			isBuiltIn: true,
		},
		asTypesKey("apiextensions.k8s.io", "CustomResourceDefinition"): {
			extractObject: defaultExtractObject,
			extractResource: func(o interface{}) (proto.Message, error) {
				if obj, ok := o.(*v1beta12.CustomResourceDefinition); ok {
					return &obj.Spec, nil
				}
				return nil, fmt.Errorf("unable to convert to v1beta1.Ingress: %T", o)
			},
			newInformer: func() (cache.SharedIndexInformer, error) {
				ext, err := p.interfaces.APIExtensionsClientset()
				if err != nil {
					return nil, err
				}
				inf := cache.NewSharedIndexInformer(
					&cache.ListWatch{
						ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
							return ext.ApiextensionsV1beta1().CustomResourceDefinitions().List(options)
						},
						WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
							return ext.ApiextensionsV1beta1().CustomResourceDefinitions().Watch(options)
						},
					},
					&v1beta12.CustomResourceDefinition{},
					0,
					cache.Indexers{})

				return inf, nil

			},
			parseJSON: func(input []byte) (interface{}, error) {
				out := &v1beta12.CustomResourceDefinition{}
				if _, _, err := deserializer.Decode(input, nil, out); err != nil {
					return nil, err
				}
				return out, nil
			},
			getStatus: noStatus,
			isEqual:   resourceVersionsMatch,
			isBuiltIn: true,
		},

		asTypesKey("apps", "Deployment"): {
			extractObject: defaultExtractObject,
			extractResource: func(o interface{}) (proto.Message, error) {
				if obj, ok := o.(*appsv1.Deployment); ok {
					return obj, nil
				}
				return nil, fmt.Errorf("unable to convert to v1.Deployment: %T", o)
			},
			newInformer: func() (cache.SharedIndexInformer, error) {
				return nil, fmt.Errorf("not implemented")
			},
			parseJSON: func(input []byte) (interface{}, error) {
				out := &appsv1.Deployment{}
				if _, _, err := deserializer.Decode(input, nil, out); err != nil {
					return nil, err
				}
				return out, nil
			},
			getStatus: noStatus,
			isEqual:   resourceVersionsMatch,
			isBuiltIn: true,
		},
	}
}

func asTypesKey(group, kind string) string {
	if group == "" {
		return kind
	}
	return fmt.Sprintf("%s/%s", group, kind)
}

func defaultExtractObject(o interface{}) metav1.Object {
	if obj, ok := o.(metav1.Object); ok {
		return obj
	}
	return nil
}

func noStatus(_ interface{}) interface{} {
	return nil
}
