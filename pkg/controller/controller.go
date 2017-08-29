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

package controller

import (
	"context"
	"fmt"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	//"k8s.io/apimachinery/pkg/api/resource"

	coredump "k8s.io/coredump-detector/apis/coredump/v1alpha1"
)

// Watcher is an example of watching on resource create/update/delete events
type CoredumpController struct {
	CoredumpClient *rest.RESTClient
	CoredumpScheme *runtime.Scheme
}

func NewCoredumpController(kubeConfig string) (*CoredumpController, error) {
	// Create the client config. Use kubeconfig if given, otherwise assume in-cluster.
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		return nil, err
	}

	// make a new config for our extension's API group, using the first config as a baseline
	exampleClient, exampleScheme, err := newCoredumpClient(config)
	if err != nil {
		return nil, err
	}

	controller := &CoredumpController{
		CoredumpClient: exampleClient,
		CoredumpScheme: exampleScheme,
	}
	return controller, nil
}

func newCoredumpClient(cfg *rest.Config) (*rest.RESTClient, *runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	if err := coredump.AddToScheme(scheme); err != nil {
		return nil, nil, err
	}

	config := *cfg
	config.GroupVersion = &coredump.SchemeGroupVersion
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: serializer.NewCodecFactory(scheme)}

	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, nil, err
	}

	return client, scheme, nil
}

// Run starts an Coredump resource controller
func (c *CoredumpController) Run(ctx context.Context) error {
	fmt.Print("Watch Coredump objects\n")

	// Watch Coredump objects
	_, err := c.watchCoredumps(ctx)
	if err != nil {
		fmt.Printf("Failed to register watch for Coredump resource: %v\n", err)
		return err
	}

	<-ctx.Done()
	return ctx.Err()
}

func (c *CoredumpController) watchCoredumps(ctx context.Context) (cache.Controller, error) {
	source := cache.NewListWatchFromClient(
		c.CoredumpClient,
		coredump.CoredumpResourcePlural,
		apiv1.NamespaceAll,
		fields.Everything())

	_, controller := cache.NewInformer(
		source,

		// The object type.
		&coredump.Coredump{},

		// resyncPeriod
		// Every resyncPeriod, all resources in the cache will retrigger events.
		// Set to 0 to disable the resync.
		0,

		// Your custom resource event handlers.
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.onAdd,
			UpdateFunc: c.onUpdate,
			DeleteFunc: c.onDelete,
		})

	go controller.Run(ctx.Done())
	return controller, nil
}

func (c *CoredumpController) onAdd(obj interface{}) {
	example := obj.(*coredump.Coredump)
	if example.Status.State != coredump.CoredumpStateCreated {
		return
	}
	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	exampleCopy := example.DeepCopy()
	message := ""
	fmt.Printf("[CONTROLLER] OnAdd %s\n", example.ObjectMeta.SelfLink)

	quotaList := coredump.CoredumpQuotaList{}
	err := c.CoredumpClient.Get().Namespace(example.ObjectMeta.Namespace).Resource(coredump.CoredumpQuotaResourcePlural).Do().Into(&quotaList)
	if err != nil {
		fmt.Printf("Error %v\n", err)
		return
	}

	exceed := false
	// check whether we exceed any quota
	for _, q := range quotaList.Items {
		totalSize := (*example.Spec.Size).DeepCopy()
		if (q.Status != coredump.QuotaStatus{}) {
			totalSize.Add(*q.Status.Used)
		}
		if totalSize.Cmp(*q.Spec.Hard) > 0 {
			exceed = true
			message = fmt.Sprintf("Quota exceed, required %s, but %s has only %s", totalSize.String(), q.ObjectMeta.Name, q.Spec.Hard.String())
		}
	}

	if exceed {
		exampleCopy.Status = coredump.CoredumpStatus{
			State:   coredump.CoredumpStateDenied,
			Message: message,
		}
		c.saveStatus(exampleCopy)
		return
	}

	// set quota
	for _, q := range quotaList.Items {
		qq := q.DeepCopy()
		qq.Status.Hard = q.Spec.Hard
		totalSize := (*example.Spec.Size).DeepCopy()
		if (q.Status != coredump.QuotaStatus{}) {
			totalSize.Add(*qq.Status.Used)
		}
		qq.Status.Used = &totalSize

		err = c.CoredumpClient.Put().
			Name(qq.ObjectMeta.Name).
			Namespace(qq.ObjectMeta.Namespace).
			Resource(coredump.CoredumpQuotaResourcePlural).
			Body(qq).
			Do().
			Error()

		if err != nil {
			fmt.Printf("%v\n", err)
		}
	}

	exampleCopy.Status = coredump.CoredumpStatus{
		State:   coredump.CoredumpStateStateAllowed,
		Message: "Ready for saving to  persistent volume",
	}
	c.saveStatus(exampleCopy)
}

func (c *CoredumpController) saveStatus(example *coredump.Coredump) {
	err := c.CoredumpClient.Put().
		Name(example.ObjectMeta.Name).
		Namespace(example.ObjectMeta.Namespace).
		Resource(coredump.CoredumpResourcePlural).
		Body(example).
		Do().
		Error()

	if err != nil {
		fmt.Printf("ERROR updating status: %v\n", err)
	} else {
		fmt.Printf("UPDATED status: %#v\n", example)
	}
}

func (c *CoredumpController) onUpdate(oldObj, newObj interface{}) {
	oldCoredump := oldObj.(*coredump.Coredump)
	newCoredump := newObj.(*coredump.Coredump)
	fmt.Printf("[CONTROLLER] OnUpdate oldObj: %s\n", oldCoredump.ObjectMeta.SelfLink)
	fmt.Printf("[CONTROLLER] OnUpdate newObj: %s\n", newCoredump.ObjectMeta.SelfLink)
}

func (c *CoredumpController) onDelete(obj interface{}) {
	example := obj.(*coredump.Coredump)
	fmt.Printf("[CONTROLLER] OnDelete %s\n", example.ObjectMeta.SelfLink)
	if example.Status.State != coredump.CoredumpStateProcessed &&
		example.Status.State != coredump.CoredumpStateFailed &&
		example.Status.State != coredump.CoredumpStateStateAllowed {
		return
	}
	// free quota for deleted coredump file
	quotaList := coredump.CoredumpQuotaList{}
	err := c.CoredumpClient.Get().Namespace(example.ObjectMeta.Namespace).Resource(coredump.CoredumpQuotaResourcePlural).Do().Into(&quotaList)
	if err != nil {
		fmt.Printf("Error %v\n", err)
		return
	}

	for _, q := range quotaList.Items {
		qq := q.DeepCopy()
		qq.Status.Hard = q.Spec.Hard
		totalSize := (*qq.Status.Used).DeepCopy()
		totalSize.Sub(*example.Spec.Size)
		qq.Status.Used = &totalSize

		err = c.CoredumpClient.Put().
			Name(qq.ObjectMeta.Name).
			Namespace(qq.ObjectMeta.Namespace).
			Resource(coredump.CoredumpQuotaResourcePlural).
			Body(qq).
			Do().
			Error()

		if err != nil {
			fmt.Printf("%v\n", err)
		}
	}
}
