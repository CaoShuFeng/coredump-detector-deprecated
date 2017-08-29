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

package main

import (
	"os"

	"github.com/golang/glog"
	"github.com/spf13/pflag"

	"k8s.io/coredump-detector/cmd/options"
	"k8s.io/coredump-detector/pkg/dump"
	"k8s.io/coredump-detector/pkg/kube"
	"k8s.io/coredump-detector/pkg/libdocker"
	"k8s.io/coredump-detector/pkg/version"
)

func main() {
	cdo := options.NewCoredumpDetectorOptions()
	po := options.NewProgressInfo()
	cdo.AddFlags(pflag.CommandLine)
	po.AddFlags(pflag.CommandLine)

	pflag.Parse()

	if cdo.PrintVersion {
		version.PrintVersion()
		os.Exit(0)
	}
	kubeClient := kube.NewClientOrDie(cdo.KubeConfig)
	dockerClient := libdocker.NewClientOrDie()

	if err := dump.Dump(kubeClient, dockerClient, po, cdo); err != nil {
		glog.Errorf(err.Error())
	}
	glog.Flush()
}
