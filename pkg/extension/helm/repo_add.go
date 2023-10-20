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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofrs/flock"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/klog/v2"
)

// RepoAdd adds a chart repository to the local repository list.
func RepoAdd(repoName, repoUrl string) error {
	// Ensure the file directory exists as it is required for file locking
	fmt.Printf("settings.RepositoryCache: %v\n", settings.RepositoryCache)
	fmt.Printf("settings.RepositoryConfig: %v\n", settings.RepositoryConfig)
	err := os.MkdirAll(filepath.Dir(settings.RepositoryConfig), os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return err
	}

	// Acquire a file lock for process synchronization
	repoFileExt := filepath.Ext(settings.RepositoryConfig)
	var lockPath string
	if len(repoFileExt) > 0 && len(repoFileExt) < len(settings.RepositoryConfig) {
		lockPath = strings.TrimSuffix(settings.RepositoryConfig, repoFileExt) + ".lock"
	} else {
		lockPath = settings.RepositoryConfig + ".lock"
	}
	fileLock := flock.New(lockPath)
	lockCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	locked, err := fileLock.TryLockContext(lockCtx, time.Second)
	if err == nil && locked {
		defer fileLock.Unlock()
	}
	if err != nil {
		return err
	}

	b, err := os.ReadFile(settings.RepositoryConfig)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	var f repo.File
	if err := yaml.Unmarshal(b, &f); err != nil {
		return err
	}

	// if o.username != "" && o.password == "" {
	// 	if o.passwordFromStdinOpt {
	// 		passwordFromStdin, err := io.ReadAll(os.Stdin)
	// 		if err != nil {
	// 			return err
	// 		}
	// 		password := strings.TrimSuffix(string(passwordFromStdin), "\n")
	// 		password = strings.TrimSuffix(password, "\r")
	// 		o.password = password
	// 	} else {
	// 		fd := int(os.Stdin.Fd())
	// 		fmt.Fprint(out, "Password: ")
	// 		password, err := term.ReadPassword(fd)
	// 		fmt.Fprintln(out)
	// 		if err != nil {
	// 			return err
	// 		}
	// 		o.password = string(password)
	// 	}
	// }

	repoEntry := repo.Entry{
		Name: repoName,
		URL:  repoUrl,
		// Username:              o.username,
		// Password:              o.password,
		// PassCredentialsAll:    o.passCredentialsAll,
		// CertFile:              o.certFile,
		// KeyFile:               o.keyFile,
		// CAFile:                o.caFile,
		// InsecureSkipTLSverify: o.insecureSkipTLSverify,
	}

	// Check if the repo name is legal
	if strings.Contains(repoName, "/") {
		return errors.Errorf("repository name (%s) contains '/', please specify a different name without '/'", repoName)
	}

	// If the repo exists do one of two things:
	// 1. If the configuration for the name is the same continue without error
	// 2. When the config is different require --force-update
	if f.Has(repoName) {
		existing := f.Get(repoName)
		if repoEntry != *existing {

			// The input coming in for the name is different from what is already
			// configured. Return an error.
			return errors.Errorf("repository name (%s) already exists, please specify a different name", repoName)
		}

		// The add is idempotent so do nothing
		klog.Infof("%q already exists with the same configuration, skipping\n", repoName)
		return nil
	}

	r, err := repo.NewChartRepository(&repoEntry, getter.All(settings))
	if err != nil {
		return err
	}

	if settings.RepositoryCache != "" {
		r.CachePath = settings.RepositoryCache
	}
	if _, err := r.DownloadIndexFile(); err != nil {
		return errors.Wrapf(err, "looks like %q is not a valid chart repository or cannot be reached", repoUrl)
	}

	f.Update(&repoEntry)

	if err := f.WriteFile(settings.RepositoryConfig, 0644); err != nil {
		return err
	}
	klog.Infof("%q has been added to your repositories\n", repoName)
	return nil
}
