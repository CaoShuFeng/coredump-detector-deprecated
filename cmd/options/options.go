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

package options

import (
	"flag"

	"github.com/spf13/pflag"
)

// CoredumpDetectorOptions contains node problem detector command line and application options.
type CoredumpDetectorOptions struct {
	// command line options
	PrintVersion bool
	KubeConfig   string
	DumpDir      string
}

// ProgressInfo contains pid info passed by kernel
// http://man7.org/linux/man-pages/man5/core.5.html
type ProgressInfo struct {
	HostPid      string // %P
	ContainerPid string // %p
	Filename     string // %e
	Time         string // %t
}

func NewCoredumpDetectorOptions() *CoredumpDetectorOptions {
	return &CoredumpDetectorOptions{}
}

func NewProgressInfo() *ProgressInfo {
	return &ProgressInfo{}
}

// AddFlags adds node problem detector command line options to pflag.
func (cdo *CoredumpDetectorOptions) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&cdo.PrintVersion, "version", false, "Print version information and quit")
	fs.StringVarP(&cdo.KubeConfig, "kubeconfig", "c", "", "path to kubeconfig file")
	fs.StringVarP(&cdo.DumpDir, "dump-dir", "d", "/var/coredump", "Directory where coredump files saved")
}

// AddFlags add progress info command line options to pflag.
func (po *ProgressInfo) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&po.HostPid, "hostPid", "P", "", "PID of dumped process, as seen in the initial PID namespace.")
	fs.StringVarP(&po.ContainerPid, "containerPid", "p", "", "PID of dumped process, as seen in the PID namespace in which the process resides")
	fs.StringVarP(&po.Filename, "filename", "e", "", "executable filename (without path prefix)")
	fs.StringVarP(&po.Time, "time", "t", "", "time of dump, expressed as seconds since the Epoch")

}

func init() {
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
}
