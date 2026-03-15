package handlers

import "time"

func formatDateOnly(value *time.Time) string {
	if value == nil {
		return ""
	}

	return value.Format("2006-01-02")
}
