/*
Copyright 2022 The Ketches Authors.

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
	"sync"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/klog/v2"
)

var errNoRepositories = errors.New("no repositories found. You must add one before updating")

func RepoUpdate(repoNames ...string) error {
	f, err := repo.LoadFile(settings.RepositoryConfig)
	switch {
	case isNotExist(err):
		return errNoRepositories
	case err != nil:
		return errors.Wrapf(err, "failed loading file: %s", settings.RepositoryConfig)
	case len(f.Repositories) == 0:
		return errNoRepositories
	}

	var repos []*repo.ChartRepository
	updateAllRepos := len(repoNames) == 0

	if !updateAllRepos {
		// Fail early if the user specified an invalid repo to update
		if err := checkRequestedRepos(repoNames, f.Repositories); err != nil {
			return err
		}
	}

	for _, cfg := range f.Repositories {
		if updateAllRepos || isRepoRequested(cfg.Name, repoNames) {
			r, err := repo.NewChartRepository(cfg, getter.All(settings))
			if err != nil {
				return err
			}
			if settings.RepositoryCache != "" {
				r.CachePath = settings.RepositoryCache
			}
			repos = append(repos, r)
		}
	}

	return updateCharts(repos, false)
}

func updateCharts(repos []*repo.ChartRepository, failOnRepoUpdateFail bool) error {
	klog.Infoln("Hang tight while we grab the latest from your chart repositories...")
	var wg sync.WaitGroup
	var repoFailList []string
	for _, re := range repos {
		wg.Add(1)
		go func(re *repo.ChartRepository) {
			defer wg.Done()
			if _, err := re.DownloadIndexFile(); err != nil {
				klog.Infof("...Unable to get an update from the %q chart repository (%s):\n\t%s\n", re.Config.Name, re.Config.URL, err)
				repoFailList = append(repoFailList, re.Config.URL)
			} else {
				klog.Infof("...Successfully got an update from the %q chart repository\n", re.Config.Name)
			}
		}(re)
	}
	wg.Wait()

	if len(repoFailList) > 0 && failOnRepoUpdateFail {
		return fmt.Errorf("failed to update the following repositories: %s", repoFailList)
	}

	klog.Infoln("Update Complete. ⎈Happy Helming!⎈")
	return nil
}

func checkRequestedRepos(requestedRepos []string, validRepos []*repo.Entry) error {
	for _, requestedRepo := range requestedRepos {
		found := false
		for _, vr := range validRepos {
			if requestedRepo == vr.Name {
				found = true
				break
			}
		}
		if !found {
			return errors.Errorf("no repositories found matching '%s'.  Nothing will be updated", requestedRepo)
		}
	}
	return nil
}

func isRepoRequested(repoName string, requestedRepos []string) bool {
	for _, requestedRepo := range requestedRepos {
		if repoName == requestedRepo {
			return true
		}
	}
	return false
}
