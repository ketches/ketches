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

package model

var (
// ErrInvalidEmail = errors.New("invalid email")
)

type SpaceUri struct {
	SpaceID string `uri:"space_id"`
}

type ApplicationUri struct {
	ApplicationID string `uri:"application_id"`
}

type PagedFilterInterface interface {
	GetPage() int
	GetLimit() int
}

type PagedFilter struct {
	PageNo   int `form:"page_no"`
	PageSize int `form:"page_size"`
}

func (f *PagedFilter) GetPage() int {
	if f.PageNo < 1 {
		return 1
	}
	return f.PageNo
}

func (f *PagedFilter) GetLimit() int {
	if f.PageSize < 1 {
		return 10
	}
	return f.PageSize
}

type QueryAndPagedFilter struct {
	Query       string `form:"query"`
	SortBy      string `form:"sort_by"`
	PagedFilter `form:",inline"`
}

// PagedResult is a helper function to return a paged result and total count and error
func PagedResult[S ~[]T, T any](items S, filter PagedFilterInterface) (S, int) {
	total := len(items)
	if total == 0 {
		return nil, total
	}

	start := (filter.GetPage() - 1) * filter.GetLimit()
	if start >= total {
		return nil, total
	}
	end := start + filter.GetLimit()
	if end > total {
		end = total
	}

	ret := items[start:end]
	return ret, total
}

type Port struct {
	Number   int32         `json:"number"`
	Gateways []PortGateway `json:"gateways"`
}

type PortGateway struct {
	Type            string         `json:"type"`
	NodePort        int32          `json:"node_port"`
	Host            string         `json:"host"`
	Path            string         `json:"path"`
	AdvancedOptions map[string]any `json:"advanced_options"`
}

type KeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
