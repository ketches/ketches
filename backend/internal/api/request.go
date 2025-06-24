package api

import "gorm.io/gorm"

type PagedFilter struct {
	PageNo   int `form:"pageNo,default=1"`
	PageSize int `form:"pageSize,default=10"`
}

type QueryAndPagedFilter struct {
	PagedFilter `form:",inline"`
	Query       string `form:"query"`
	SortBy      string `form:"sortBy,default=created_at"`
	SortOrder   string `form:"sortOrder,default=DESC"`
}

func (f *QueryAndPagedFilter) PagedSQL(db *gorm.DB) *gorm.DB {
	if f.PageNo < 1 {
		f.PageNo = 1
	}
	if f.PageSize < 1 {
		f.PageSize = 10
	}
	return db.Offset((f.PageNo - 1) * f.PageSize).Limit(f.PageSize).Order(f.SortBy + " " + f.SortOrder)
}

// PagedResult is a helper function to return a paged result and total count and error
func PagedResult[S ~[]T, T any](items S, pagedFilter *PagedFilter) (S, int) {
	total := len(items)
	if total == 0 {
		return nil, total
	}

	start := (pagedFilter.PageNo - 1) * pagedFilter.PageSize
	if start >= total {
		return nil, total
	}
	end := min(start+pagedFilter.PageSize, total)

	ret := items[start:end]
	return ret, total
}

type KeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
