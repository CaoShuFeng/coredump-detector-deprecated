/*
Copyright 2018 The Kubernetes Authors All rights reserved.

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

package libdocker

import (
	"time"

	"github.com/golang/glog"

	internalapi "k8s.io/kubernetes/pkg/kubelet/apis/cri"
	runtimeapi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
	"k8s.io/kubernetes/pkg/kubelet/remote"
)

type CRIClient struct {
	CRIRuntimeClient internalapi.RuntimeService
}

func ListAllContainers(rs internalapi.RuntimeService) ([]*runtimeapi.Container, error) {
	filter := &runtimeapi.ContainerFilter{}
	containers, err := rs.ListContainers(filter)
	if err != nil {
                glog.Errorf("rc.ListContainer failed: %v", err)
		return nil, err
        }

	return containers, nil
}

func GetPodStatus (rs internalapi.RuntimeService, podSandboxID string) (*runtimeapi.PodSandboxStatus, error) {
	PodSandboxStatus, err := rs.PodSandboxStatus(podSandboxID)
	if err != nil {
                glog.Errorf("rc.GetPodStatus failed: %v", err)
                return nil, err
	}

	return PodSandboxStatus, nil
}

func NewCRIClientOrDie() CRIClient {
	rService, err := remote.NewRemoteRuntimeService("unix:///var/run/dockershim.sock", 300*time.Second)
	if err != nil {
		glog.Errorf("NewRemoteRuntimeService failed: %v", err)
	}

	return CRIClient{
		CRIRuntimeClient: rService,
	}

}
