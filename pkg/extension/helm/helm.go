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

package helm

import (
	"fmt"
	"log"
	"os"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/registry"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
)

var (
	settings = cli.New()
)

func actionConfiguration(restConfig *rest.Config, namespace string) *action.Configuration {
	actionConfig := new(action.Configuration)

	cliConfig := genericclioptions.NewConfigFlags(false)
	cliConfig.APIServer = &restConfig.Host
	cliConfig.BearerToken = &restConfig.BearerToken
	cliConfig.Namespace = &namespace
	wrapper := func(*rest.Config) *rest.Config {
		return restConfig
	}
	cliConfig.WrapConfigFn = wrapper

	actionConfig.Init(cliConfig, namespace, os.Getenv("HELM_DRIVER"), debug)
	registryClient, _ := registry.NewClient(
		registry.ClientOptDebug(settings.Debug),
		registry.ClientOptEnableCache(true),
		registry.ClientOptWriter(os.Stderr),
		registry.ClientOptCredentialsFile(settings.RegistryConfig),
	)

	actionConfig.RegistryClient = registryClient
	return actionConfig
}

func debug(format string, v ...interface{}) {
	if settings.Debug {
		format = fmt.Sprintf("[debug] %s\n", format)
		log.Output(2, fmt.Sprintf(format, v...))
	}
}

func warning(format string, v ...interface{}) {
	format = fmt.Sprintf("WARNING: %s\n", format)
	fmt.Fprintf(os.Stderr, format, v...)
}

func isNotExist(err error) bool {
	return os.IsNotExist(errors.Cause(err))
}
