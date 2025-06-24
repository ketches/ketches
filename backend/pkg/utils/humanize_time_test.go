package utils

import (
	"testing"
	"time"

	"github.com/dustin/go-humanize"
)

type testList []struct {
	name, got, exp string
}

func (tl testList) validate(t *testing.T) {
	for _, test := range tl {
		if test.got != test.exp {
			t.Errorf("On %v, expected '%v', but got '%v'",
				test.name, test.exp, test.got)
		}
	}
}

func TestPast(t *testing.T) {
	now := time.Now()
	testdata := []struct {
		name string
		then time.Time
		exp  string
	}{
		{"刚刚", now, "刚刚"},
		{"1 秒前", now.Add(-1 * time.Second), "1 秒前"},
		{"12 秒前", now.Add(-12 * time.Second), "12 秒前"},
		{"30 秒前", now.Add(-30 * time.Second), "30 秒前"},
		{"45 秒前", now.Add(-45 * time.Second), "45 秒前"},
		{"1 分钟前", now.Add(-63 * time.Second), "1 分钟前"},
		{"15 分钟前", now.Add(-15 * time.Minute), "15 分钟前"},
		{"1 小时前", now.Add(-63 * time.Minute), "1 小时前"},
		{"2 小时前", now.Add(-2 * time.Hour), "2 小时前"},
		{"21 小时前", now.Add(-21 * time.Hour), "21 小时前"},
		{"1 天前", now.Add(-26 * time.Hour), "1 天前"},
		{"2 天前", now.Add(-49 * time.Hour), "2 天前"},
		{"3 天前", now.Add(-3 * humanize.Day), "3 天前"},
		{"1 周前 (1)", now.Add(-7 * humanize.Day), "1 周前"},
		{"1 周前 (2)", now.Add(-12 * humanize.Day), "1 周前"},
		{"2 周前", now.Add(-15 * humanize.Day), "2 周前"},
		{"1 个月前", now.Add(-39 * humanize.Day), "1 个月前"},
		{"3 个月前", now.Add(-99 * humanize.Day), "3 个月前"},
		{"1 年前 (1)", now.Add(-365 * humanize.Day), "1 年前"},
		{"1 年前 (1)", now.Add(-400 * humanize.Day), "1 年前"},
		{"2 年前 (1)", now.Add(-548 * humanize.Day), "2 年前"},
		{"2 年前 (2)", now.Add(-725 * humanize.Day), "2 年前"},
		{"2 年前 (3)", now.Add(-800 * humanize.Day), "2 年前"},
		{"3 年前", now.Add(-3 * humanize.Year), "3 年前"},
		{"很久以前", now.Add(-humanize.LongTime), "很久以前"},
	}

	for _, test := range testdata {
		got := HumanizeTime(test.then)
		if got != test.exp {
			t.Errorf("On %v, expected '%v', but got '%v'",
				test.name, test.exp, got)
		}
	}
}
