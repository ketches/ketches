package gutil

import (
	"testing"
)

func TestShuffle(t *testing.T) {
	in := []string{"Hello", "Jay", "World", "Pesco", "Chou", "Ding"}
	Shuffle(in)
	t.Log(in)
}

func TestReverse(t *testing.T) {
	in := []string{"Hello", "Jay", "World", "Pesco", "Chou", "Ding"}
	Reverse(in)
	t.Log(in)
}
