# Copyright 2017 The Kubernetes Authors All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM ubuntu
MAINTAINER Cao Shufeng <caosf.fnst@cn.fujitsu.com>

RUN apt update
RUN apt install file -y
ADD ./bin/coredump-detector /coredump-detector
ADD ./bin/kubectl /bin/kubectl
ADD ./detector-script.sh /detector-script.sh
ADD ./config /config
