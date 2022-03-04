#!/usr/bin/env bats

#
# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

start_adapter() {
  run systemctl start etcd-adapter
  [ "$status" -eq 0 ]
  sleep "$1"
}

stop_adapter() {
  run systemctl stop etcd-adapter
  [ "$status" -eq 0 ]
  sleep "$1"
}

### Test Case
#pre
@test "Build etcd-adapter" {
  run go build -o ./etcd-adapter ./main.go
  [ "$status" -eq 0 ]

  mkdir -p /usr/local/etcd-adapter
  cp ./etcd-adapter /usr/local/etcd-adapter
  cp ./test/shell/etcd-adapter.service /usr/lib/systemd/system/etcd-adapter.service
  run systemctl daemon-reload
  [ "$status" -eq 0 ]
}

#1
@test "Run etcd-adapter" {
  start_adapter 1
  stop_adapter 0
}

#post
@test "Clean environment" {
  # stop dashboard service
  stop_adapter 0

  # clean configure and log files
  rm -rf /usr/local/etcd-adapter
  rm /usr/lib/systemd/system/etcd-adapter.service

  # reload systemd services
  run systemctl daemon-reload
  [ "$status" -eq 0 ]
}
