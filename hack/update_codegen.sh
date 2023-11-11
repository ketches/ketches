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

# package name for generated code. eg: github.com/foo/bar
ROOT_PKG="github.com/ketches/ketches"

# path where API type definitions are stored
API_PKG="${ROOT_PKG}/api"

# api group versions to gen code for. format: "group1/version1,group2/version2"
GROUP_VERSIONS="ci/v1alpha1,core/v1alpha1"

# path under pkg/ where generated code will be saved
OUTPUT_PKG="${ROOT_PKG}/pkg/generated"

# path to the boilerplate file to use for code generation
BOILERPLATE="./hack/boilerplate.go.txt"

# generators to run. available generators: deepcopy, defaulter, applyconfiguration, client, lister, informer
generators=(
#  "deepcopy" # deepcopy-gen is deprecated, use controller-gen instead for faster generation.
  "defaulter"
#   "applyconfiguration"
   "client"
   "informer"
   "lister"
)

# list of import paths to get input types from.
INPUT_DIRS=""
# fill INPUT_DIRS from GROUP_VERSIONS
for gv in $(echo ${GROUP_VERSIONS} | tr "," "\n"); do
  INPUT_DIRS="${INPUT_DIRS},${API_PKG}/${gv}"
done
INPUT_DIRS=$(echo ${INPUT_DIRS} | cut -c 2-)

deepcopy_gen() {
  echo " ⚠ Deprecated: deepcopy-gen is deprecated, use controller-gen instead for faster generation."
  deepcopy-gen \
    --input-dirs ${INPUT_DIRS} \
    --output-base . \
    --output-file-base zz_generated.deepcopy \
    --trim-path-prefix ${ROOT_PKG} \
    --go-header-file ${BOILERPLATE}
}

defaulter_gen() {
  echo " ⚠ Deprecated: defaulter-gen is deprecated, use kubebuilder create webhook --defaulting instead."
  defaulter-gen \
    --input-dirs ${INPUT_DIRS} \
    --output-base . \
    --output-file-base zz_generated.defaults \
    --trim-path-prefix ${ROOT_PKG} \
    --go-header-file ${BOILERPLATE}
}

applyconfiguration_gen() {
  applyconfiguration-gen \
    --input-dirs ${INPUT_DIRS} \
    --output-base . \
    --output-package ${OUTPUT_PKG}/applyconfiguration \
    --trim-path-prefix ${ROOT_PKG} \
    --go-header-file ${BOILERPLATE}
}

client_gen() {
  client-gen \
    --input-base ${ROOT_PKG}/api \
    --input ${GROUP_VERSIONS} \
    --output-base . \
    --output-package ${OUTPUT_PKG}/clientset \
    --clientset-name versioned \
    --trim-path-prefix ${ROOT_PKG} \
    --go-header-file ${BOILERPLATE}
}

informer_gen() {
  informer-gen \
    --input-dirs ${INPUT_DIRS} \
    --versioned-clientset-package ${OUTPUT_PKG}/clientset/versioned \
    --listers-package ${OUTPUT_PKG}/listers \
    --output-base . \
    --output-package ${OUTPUT_PKG}/informers \
    --trim-path-prefix ${ROOT_PKG} \
    --go-header-file ${BOILERPLATE}
}

lister_gen() {
  lister-gen \
    --input-dirs ${INPUT_DIRS} \
    --output-base . \
    --output-package ${OUTPUT_PKG}/listers \
    --trim-path-prefix ${ROOT_PKG} \
    --go-header-file ${BOILERPLATE}
}


for generator in "${generators[@]}"; do
  # install tools if not present
  if ! command -v "${generator}-gen" &> /dev/null; then
    echo " ⚒ installing ${generator}-gen ..."
    go install k8s.io/code-generator/cmd/${generator}-gen@latest
    echo " ✔ ${generator}-gen installed"
  fi

  echo " ⚒ ${generator}-gen for ${GROUP_VERSIONS} ..."
  # 调用函数
  ${generator}_gen
  echo " ✔ ${generator}-gen done"
done