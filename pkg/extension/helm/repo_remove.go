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
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/helmpath"
	"helm.sh/helm/v3/pkg/repo"
)

// RepoRemove removes a repository from the repositories file.
func RepoRemove(repoName string) error {
	r, err := repo.LoadFile(settings.RepositoryConfig)
	if isNotExist(err) || len(r.Repositories) == 0 {
		return errors.New("no repositories configured")
	}
	// for _, name := range o.names {
	if !r.Remove(repoName) {
		return errors.Errorf("no repo named %q found", repoName)
	}
	if err := r.WriteFile(settings.RepositoryConfig, 0600); err != nil {
		return err
	}

	idx := filepath.Join(settings.RepositoryCache, helmpath.CacheChartsFile(repoName))
	if _, err := os.Stat(idx); err == nil {
		os.Remove(idx)
	}

	idx = filepath.Join(settings.RepositoryCache, helmpath.CacheIndexFile(repoName))
	if _, err := os.Stat(idx); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return errors.Wrapf(err, "can't remove index file %s", idx)
	}
	err = os.Remove(idx)
	if err != nil {
		return err
	}
	fmt.Printf("%q has been removed from your repositories\n", repoName)
	return nil
}
