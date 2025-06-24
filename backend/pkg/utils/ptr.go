package utils

func Ptr[V any](v V) *V {
	return &v
}
