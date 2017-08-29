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

package dump

import (
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	coredump "k8s.io/coredump-detector/apis/coredump/v1alpha1"
	"k8s.io/coredump-detector/cmd/options"
	"k8s.io/coredump-detector/pkg/apiextensions"
	"k8s.io/coredump-detector/pkg/kube"
	"k8s.io/coredump-detector/pkg/libdocker"

	"github.com/docker/docker/api/types"
	"github.com/golang/glog"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apitypes "k8s.io/apimachinery/pkg/types"
)

type DumpInfo struct {
	ContainerName string
	Pod           string
	Namespace     string
	Uid           string
	Pid           string
	Filename      string
	Time          string
}

func Dump(kc kube.Client, dc libdocker.Client, progressInfo *options.ProgressInfo, options *options.CoredumpDetectorOptions) error {
	if progressInfo.ContainerPid == progressInfo.HostPid {
		return saveOthers(progressInfo, options)
	}
	containers, err := dc.ContainerList(types.ContainerListOptions{})
	if err != nil {
		return err
	}
	for _, c := range containers {
		for _, name := range c.Names {
			// a k8s container
			// format of container name:
			// https://github.com/kubernetes/kubernetes/blob/v1.8.0-beta.1/pkg/kubelet/dockershim/naming.go
			if strings.HasPrefix(name, "/k8s") {
				body, err := dc.ContainerTop(c.ID)
				if err != nil {
					return err
				}
				index := 0
				// get the index of PID
				for i, t := range body.Titles {
					if strings.EqualFold(t, "PID") {
						index = i
					}
				}
				for _, p := range body.Processes {
					if p[index] == progressInfo.HostPid {
						//a progress in k8s pod.
						// get pod's info from kubernetes cluster
						dumpInfo, _ := parseContainerName(name, progressInfo)
						ok, err := validate(dumpInfo, kc)
						if err != nil {
							return err
						}
						if !ok {
							glog.Info("can not find pod info from kube-apiserver")
							return nil
						}
						size, err := save(dumpInfo, options)
						if err != nil {
							return err
						}
						return saveToApiServer(dumpInfo, options, size)
					}
				}

			}
		}
	}
	return saveOthers(progressInfo, options)
}

// saveOthers saves coredump files in host.
func saveOthers(progressInfo *options.ProgressInfo, options *options.CoredumpDetectorOptions) error {
	dirname := path.Join(options.DumpDir, "others")
	if err := os.MkdirAll(dirname, 0775); err != nil {
		return err
	}
	filename := progressInfo.Filename + "-" + progressInfo.HostPid + "-" + progressInfo.Time
	file, err := os.Create(path.Join(dirname, filename))
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := io.Copy(file, os.Stdin); err != nil {
		return err
	}
	glog.Infof("Saved dumpfile at: %s\n", file.Name())
	return nil
}

func parseContainerName(name string, progressInfo *options.ProgressInfo) (*DumpInfo, error) {
	// Docker adds a "/" prefix to names. so trim it.
	name = strings.TrimPrefix(name, "/")

	parts := strings.Split(name, "_")
	// Tolerate the random suffix.
	// TODO: Remove 7 field case when docker 1.11 is deprecated.
	if len(parts) != 6 && len(parts) != 7 {
		return nil, fmt.Errorf("failed to parse the container name: %q", name)
	}
	return &DumpInfo{
		ContainerName: parts[1],
		Pod:           parts[2],
		Namespace:     parts[3],
		Uid:           parts[4],
		Pid:           progressInfo.HostPid,
		Filename:      progressInfo.Filename,
		Time:          progressInfo.Time,
	}, nil
}

func save(dumpInfo *DumpInfo, options *options.CoredumpDetectorOptions) (int64, error) {
	dirname := path.Join(options.DumpDir, dumpInfo.Namespace, dumpInfo.Pod+"-"+dumpInfo.Uid, dumpInfo.ContainerName)
	if err := os.MkdirAll(dirname, 0775); err != nil {
		return 0, err
	}
	filename := "coredump-" + dumpInfo.Filename + "-" + dumpInfo.Pod + "-" + dumpInfo.Time
	file, err := os.Create(path.Join(dirname, filename))
	if err != nil {
		return 0, err
	}
	defer file.Close()
	size, err := io.Copy(file, os.Stdin)
	if err != nil {
		return 0, err
	}
	glog.Infof("Saved dumpfile at: %s\n", file.Name())
	return size, nil
}

// validate validate the pod info with the kube-apiserver.
func validate(dumpInfo *DumpInfo, kc kube.Client) (bool, error) {
	pod, err := kc.GetPod(dumpInfo.Namespace, dumpInfo.Pod)
	if err != nil {
		return false, err
	}

	// validate UID
	if string(pod.ObjectMeta.UID) != dumpInfo.Uid {
		return false, nil
	}
	// validate container name
	for _, c := range pod.Spec.Containers {
		if c.Name == dumpInfo.ContainerName {
			return true, nil
		}
	}
	return false, nil
}

func saveToApiServer(dumpInfo *DumpInfo, cdo *options.CoredumpDetectorOptions, size int64) error {
	apiextensionsClient := apiextensions.NewClientOrDie(cdo.KubeConfig)
	_, err := apiextensionsClient.CreateCoredumpDefinition()
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	coredumpClient := apiextensions.NewCoredumpClientOrDie(cdo.KubeConfig)
	pid, _ := strconv.Atoi(dumpInfo.Pid)
	dumptime, _ := strconv.ParseInt(dumpInfo.Time, 10, 64)
	cd := &coredump.Coredump{
		ObjectMeta: metav1.ObjectMeta{
			Name: "coredump-" + dumpInfo.Filename + "-" + dumpInfo.Pod + "-" + dumpInfo.Time,
		},
		Spec: coredump.CoredumpSpec{
			ContainerName: dumpInfo.ContainerName,
			Pod:           dumpInfo.Pod,
			Uid:           apitypes.UID(dumpInfo.Uid),
			Pid:           pid,
			Filename:      dumpInfo.Filename,
			Time:          metav1.NewTime(time.Unix(dumptime, 0)),
			Volume:        "",
			Size:          resource.NewQuantity(size, resource.BinarySI),
		},
		Status: coredump.CoredumpStatus{
			State:   coredump.CoredumpStateCreated,
			Message: "Created, not saved yet, need to check quota and then save it to persistent volume",
		},
	}
	_, err = coredumpClient.CreateCoredump(cd, dumpInfo.Namespace)
	return err
}
