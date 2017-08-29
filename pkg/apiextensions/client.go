/*
Copyright 2017 The Kubernetes Authors All rights reserved.

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

package apiextensions

import (
	"fmt"
	"reflect"
	"time"

	"github.com/golang/glog"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	coredump "k8s.io/coredump-detector/apis/coredump/v1alpha1"
)

type CrdClient interface {
	CreateCoredumpDefinition() (*apiextensionsv1beta1.CustomResourceDefinition, error)
	CreateCoredumpQuotaDefinition() (*apiextensionsv1beta1.CustomResourceDefinition, error)
}

type crdClient struct {
	clientset apiextensionsclient.Interface
}

func NewClientOrDie(kubeConfig string) CrdClient {
	c := &crdClient{}
	c.clientset = newClientsetOrDie(kubeConfig)
	return c
}

func newClientsetOrDie(kubeConfig string) *apiextensionsclient.Clientset {
	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		glog.Errorf(err.Error())
	}

	// create the clientset
	clientset, err := apiextensionsclient.NewForConfig(config)
	if err != nil {
		glog.Errorf(err.Error())
	}
	return clientset
}

const exampleCRDName = coredump.CoredumpResourcePlural + "." + coredump.GroupName
const exampleCRDQuotaName = coredump.CoredumpQuotaResourcePlural + "." + coredump.GroupName

func (c *crdClient) CreateCoredumpDefinition() (*apiextensionsv1beta1.CustomResourceDefinition, error) {
	crd := &apiextensionsv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: exampleCRDName,
		},
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   coredump.GroupName,
			Version: coredump.SchemeGroupVersion.Version,
			Scope:   apiextensionsv1beta1.NamespaceScoped,
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Plural: coredump.CoredumpResourcePlural,
				Kind:   reflect.TypeOf(coredump.Coredump{}).Name(),
			},
		},
	}
	_, err := c.clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
	if err != nil {
		return nil, err
	}

	// wait for CRD being established
	err = wait.Poll(500*time.Millisecond, 60*time.Second, func() (bool, error) {
		crd, err = c.clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Get(exampleCRDName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		for _, cond := range crd.Status.Conditions {
			switch cond.Type {
			case apiextensionsv1beta1.Established:
				if cond.Status == apiextensionsv1beta1.ConditionTrue {
					return true, err
				}
			case apiextensionsv1beta1.NamesAccepted:
				if cond.Status == apiextensionsv1beta1.ConditionFalse {
					fmt.Printf("Name conflict: %v\n", cond.Reason)
				}
			}
		}
		return false, err
	})
	if err != nil {
		deleteErr := c.clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(exampleCRDName, nil)
		if deleteErr != nil {
			return nil, errors.NewAggregate([]error{err, deleteErr})
		}
		return nil, err
	}
	return crd, nil
}

func (c *crdClient) CreateCoredumpQuotaDefinition() (*apiextensionsv1beta1.CustomResourceDefinition, error) {
	crd := &apiextensionsv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: exampleCRDQuotaName,
		},
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   coredump.GroupName,
			Version: coredump.SchemeGroupVersion.Version,
			Scope:   apiextensionsv1beta1.NamespaceScoped,
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Plural: coredump.CoredumpQuotaResourcePlural,
				Kind:   reflect.TypeOf(coredump.CoredumpQuota{}).Name(),
			},
		},
	}
	_, err := c.clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
	if err != nil {
		return nil, err
	}

	// wait for CRD being established
	err = wait.Poll(500*time.Millisecond, 60*time.Second, func() (bool, error) {
		crd, err = c.clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Get(exampleCRDQuotaName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		for _, cond := range crd.Status.Conditions {
			switch cond.Type {
			case apiextensionsv1beta1.Established:
				if cond.Status == apiextensionsv1beta1.ConditionTrue {
					return true, err
				}
			case apiextensionsv1beta1.NamesAccepted:
				if cond.Status == apiextensionsv1beta1.ConditionFalse {
					fmt.Printf("Name conflict: %v\n", cond.Reason)
				}
			}
		}
		return false, err
	})
	if err != nil {
		deleteErr := c.clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(exampleCRDQuotaName, nil)
		if deleteErr != nil {
			return nil, errors.NewAggregate([]error{err, deleteErr})
		}
		return nil, err
	}
	return crd, nil
}

type CoredumpClient interface {
	CreateCoredump(*coredump.Coredump, string) (*coredump.Coredump, error)
}

type coredumpClient struct {
	clientset *rest.RESTClient
}

func NewCoredumpClientOrDie(kubeConfig string) CoredumpClient {
	c := &coredumpClient{}
	c.clientset = newCoredumpClientsetOrDie(kubeConfig)
	return c
}

func newCoredumpClientsetOrDie(kubeConfig string) *rest.RESTClient {
	scheme := runtime.NewScheme()
	if err := coredump.AddToScheme(scheme); err != nil {
		glog.Errorf(err.Error())
	}

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		glog.Errorf(err.Error())
	}

	// create the clientset
	config.GroupVersion = &coredump.SchemeGroupVersion
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: serializer.NewCodecFactory(scheme)}

	client, err := rest.RESTClientFor(config)
	if err != nil {
		glog.Errorf(err.Error())
	}

	return client
}

func (c *coredumpClient) CreateCoredump(cd *coredump.Coredump, namespace string) (*coredump.Coredump, error) {
	var result coredump.Coredump
	err := c.clientset.Post().
		Resource(coredump.CoredumpResourcePlural).
		Namespace(namespace).
		Body(cd).
		Do().Into(&result)
	return &result, err
}
