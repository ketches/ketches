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

package clusterset

import (
	"golang.org/x/exp/maps"
	"sync"
)

type Clusterset interface {
	List(opts ListOptions) []Cluster
	Set(key string, value Cluster)
	Cluster(name string) (Cluster, bool)
	Forget(key string)
}

type ListOptions struct {
}

var _ Clusterset = (*clusterset)(nil)

func NewClusterset() Clusterset {
	return &clusterset{
		m: make(map[string]Cluster),
	}
}

type clusterset struct {
	sync.Mutex
	m map[string]Cluster
}

func (cs *clusterset) List(opts ListOptions) []Cluster {
	maps.Values(cs.m)
	cs.Lock()
	defer cs.Unlock()
	clusters := maps.Values(cs.m)
	return clusters
}

func (cs *clusterset) Set(key string, value Cluster) {
	cs.Lock()
	defer cs.Unlock()
	cs.m[key] = value
}

func (cs *clusterset) Cluster(key string) (Cluster, bool) {
	cs.Lock()
	defer cs.Unlock()
	value, ok := cs.m[key]
	return value, ok
}

func (cs *clusterset) Forget(key string) {
	cs.Lock()
	defer cs.Unlock()
	delete(cs.m, key)
}
