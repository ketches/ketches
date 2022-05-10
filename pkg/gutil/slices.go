package gutil

import (
	"math/rand"
	"time"
)

type tbase interface {
	int8 | int16 | int32 | int64 | int | uint8 | uint16 | uint32 | uint64 | uint | float32 | float64 | string | bool | complex64 | complex128
}

func Contains[T tbase](slices []T, e T) bool {
	for _, a := range slices {
		if a == e {
			return true
		}
	}
	return false
}

func Distinct[T tbase](slices []T) []T {
	var keys map[T]bool
	for _, e := range slices {
		keys[e] = true
	}
	result := make([]T, len(keys))
	for k := range keys {
		result = append(result, k)
	}
	return result
}

func Shuffle[T tbase](s []T) {
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := len(s) - 1; i > 0; i-- {
		j := random.Intn(i + 1)
		s[i], s[j] = s[j], s[i]
	}
}

func Reverse[T tbase](s []T) {
	l := len(s)
	for i := l/2 - 1; i >= 0; i-- {
		s[i], s[l-i-1] = s[l-i-1], s[i]
	}
}
