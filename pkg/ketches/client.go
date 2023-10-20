/*
Copyright 2023 The Ketches Authors.

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

package ketches

import (
	"github.com/ketches/ketches/pkg/generated/clientset/versioned"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	inClusterClient versioned.Interface
)

func Client() versioned.Interface {
	if inClusterClient == nil {
		inClusterClient = versioned.NewForConfigOrDie(ctrl.GetConfigOrDie())
	}
	return inClusterClient
}

func ClientFromConfig(config *rest.Config) (versioned.Interface, error) {
	return versioned.NewForConfig(config)
}
