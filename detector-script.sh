#!/bin/bash
#
#Copyright 2017 The Kubernetes Authors All rights reserved.
#
#Licensed under the Apache License, Version 2.0 (the "License");
#you may not use this file except in compliance with the License.
#You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
#Unless required by applicable law or agreed to in writing, software
#distributed under the License is distributed on an "AS IS" BASIS,
#WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#See the License for the specific language governing permissions and
#limitations under the License.


# This is a script runs which needs to run inside privileged container.
# This script does the following things:
# 1) cp coredump-detector binary to host
# 2) set kubeconfig for coredump-detector
# 3) set kernel.core_pattern
# 4) mv core dump files to persistent volume

set -x

# save saves coredump file to persistent volume
save() {
	while read f; do
		t=`file -b -i $f`
		if [ "$t" = "application/x-coredump; charset=binary" ]; then
			saveToPersistentVolume $f
		fi
	done
}

# /pv is a persistent volume in kubernetes cluster
saveToPersistentVolume() {
	d=`dirname $1`
	if [ "$d" = "/var/coredump/others" ]; then
		# coredump files out of k8s cluster
		return
	fi
	dest=/pv/${d:14}
	coredump=`basename $1`
	namespace=`echo $1 | tr "/" "\n" | head -n 4 | tail -n 1`
	state=`kubectl get coredump $coredump  -o go-template={{.status.state}} -n=$namespace`
	if [ $? -eq 0 ]
	then
		if [ "$state" = "Allowed" ]; then
			mkdir -p $dest
			# we need to do tenant isolation for dump files, like using nfs access
			# permissions, or publish core files in web application. 
			mv $1 $dest
			# set status
			kubectl patch coredump $coredump -p  '{"status":{"message":"Saved to persistent volume","state":"Saved"}}' --type='merge' -n $namespace
			# set persistent volume: pv-name:path
			kubectl patch coredump $coredump -p  '{"spec":{"volume":"nfs:'${d:13}'"}}' --type='merge' -n $namespace
		fi
	else
		# this should never happen
		echo "Not found in apiserver, removed $1"
		rm $1
	fi
}

# start container with -v /coredump/:/coredump
cp /coredump-detector /coredump/

server="https:\/\/${KUBERNETES_PORT_443_TCP_ADDR}:${KUBERNETES_SERVICE_PORT}"
sed -i "s/@SERVER_PORT@/${server}/g" /config
token=`cat /run/secrets/kubernetes.io/serviceaccount/token`
sed -i "s/@TOKEN@/${token}/g" /config

cp /config /coredump
cp /run/secrets/kubernetes.io/serviceaccount/ca.crt /coredump/
echo "|/coredump/coredump-detector -P=%P -p=%p -e=%e -t=%t -c=/coredump/config --log_dir=/coredump/ --v=10" > /proc/sys/kernel/core_pattern

# start container with -v /var/coredump/:/var/coredump
while true
do
	find /var/coredump/ -type f -mmin +4 | save
	sleep 60
done
