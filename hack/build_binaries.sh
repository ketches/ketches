#!/usr/bin/env bash

# Copyright 2023 The Ketches Authors.
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

set -o errexit
set -o nounset
set -o pipefail

build_items=(
  "controller-manager"
  "api-server"
)

build_platforms=(
  "linux/amd64"
  "linux/arm64"
)

build_binary() {
  local target=$1
  for platform in "${build_platforms[@]}"; do
    platform_split=(${platform//\// })
    local goos=${platform_split[0]}
    local goarch=${platform_split[1]}
    echo "Building for ${goos}/${goarch}"
    CGO_ENABLED=0 GOOS=${goos} GOARCH=${goarch} go build -o .out/${target}/${platform}/ketches-$1 ./cmd/${target}/main.go
  done
}

if [[ $# -eq 0 ]]; then
  for item in "${build_items[@]}"; do
    build_binary $item
  done
elif [[ $# -eq 1 ]]; then
  build_binary $1
else
  echo "Usage: $0 [target]"
  echo "  target: ${build_items[@]}"
  exit 1
fi


