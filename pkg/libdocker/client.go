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

package libdocker

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
)

type Client interface {
	ContainerList(options types.ContainerListOptions) ([]types.Container, error)
	ContainerTop(containerID string) (container.ContainerTopOKBody, error)
}

type dockerClient struct {
	cli *client.Client
}

func (c dockerClient) ContainerList(options types.ContainerListOptions) ([]types.Container, error) {
	ctx := context.Background()
	return c.cli.ContainerList(ctx, options)
}

func (c dockerClient) ContainerTop(containerID string) (container.ContainerTopOKBody, error) {
	ctx := context.Background()
	return c.cli.ContainerTop(ctx, containerID, nil)
}

func NewClientOrDie() Client {
	//of course we only support dockerd running in localhost.
	cli, err := client.NewClient("unix:///var/run/docker.sock", "", nil, nil)
	if err != nil {
		panic(err)
	}
	return &dockerClient{cli: cli}
}
