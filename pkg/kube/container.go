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

package kube

import (
	"bytes"
	"context"
	v1 "k8s.io/api/core/v1"
)

type GetContainerOptions struct {
	Follow    bool
	TailLines *int64
	Previous  bool
}

func GetContainerLogs(ctx context.Context, namespace, pod, container string, opt GetContainerOptions) ([]byte, error) {
	rc, err := Client().CoreV1().Pods(namespace).GetLogs(pod, &v1.PodLogOptions{
		Container: container,
		Follow:    opt.Follow,
		TailLines: opt.TailLines,
		Previous:  opt.Previous,
	}).Stream(ctx)
	if err != nil {
		return nil, err
	}

	defer rc.Close()
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(rc)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
