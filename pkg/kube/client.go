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

package kube

import (
	"github.com/golang/glog"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type Client interface {
	GetPod(namespace, name string) (ret *v1.Pod, err error)
}

type kubeClient struct {
	clientset *kubernetes.Clientset
}

func NewClientOrDie(kubeConfig string) Client {
	c := &kubeClient{}
	c.clientset = newClientsetOrDie(kubeConfig)
	return c
}

func (c *kubeClient) GetPod(namespace, name string) (ret *v1.Pod, err error) {
	return c.clientset.CoreV1().Pods(namespace).Get(name, metav1.GetOptions{})
}

func newClientsetOrDie(kubeConfig string) *kubernetes.Clientset {
	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		glog.Errorf(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Errorf(err.Error())
	}
	return clientset

}
