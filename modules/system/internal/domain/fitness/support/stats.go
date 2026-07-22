package support

import (
	"sort"
	"time"

	fitnessResp "platform/modules/system/internal/models/response"
)

func Location(name string) *time.Location {
	location, err := time.LoadLocation(name)
	if err != nil {
		return time.FixedZone("CST", 8*60*60)
	}
	return location
}

func Today(location *time.Location) string {
	return time.Now().In(location).Format(time.DateOnly)
}

func MonthBounds(month string, location *time.Location) (time.Time, time.Time, error) {
	start, err := time.ParseInLocation("2006-01", month, location)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return start, start.AddDate(0, 1, 0), nil
}

func CalculateStats(dates []string, durations []int, now time.Time) fitnessResp.Stats {
	unique := make(map[string]struct{}, len(dates))
	monthPrefix := now.Format("2006-01")
	stats := fitnessResp.Stats{}
	for i, date := range dates {
		unique[date] = struct{}{}
		if date >= monthPrefix+"-01" && date <= now.Format(time.DateOnly) {
			stats.MonthCount++
		}
		if i < len(durations) {
			stats.TotalMinutes += durations[i]
		}
	}
	stats.TotalCount = len(unique)
	if len(unique) == 0 {
		return stats
	}

	ordered := make([]string, 0, len(unique))
	for date := range unique {
		ordered = append(ordered, date)
	}
	sort.Strings(ordered)
	run := 0
	for index, date := range ordered {
		if index == 0 {
			run = 1
		} else {
			current, _ := time.ParseInLocation(time.DateOnly, date, now.Location())
			previous, _ := time.ParseInLocation(time.DateOnly, ordered[index-1], now.Location())
			if current.Sub(previous) == 24*time.Hour {
				run++
			} else {
				run = 1
			}
		}
		if run > stats.LongestStreak {
			stats.LongestStreak = run
		}
	}

	cursor := now
	today := now.Format(time.DateOnly)
	if _, exists := unique[today]; !exists {
		cursor = now.AddDate(0, 0, -1)
	}
	for {
		if _, exists := unique[cursor.Format(time.DateOnly)]; !exists {
			break
		}
		stats.CurrentStreak++
		cursor = cursor.AddDate(0, 0, -1)
	}
	return stats
}

func ValidReminderTime(value string) bool {
	_, err := time.Parse("15:04", value)
	return err == nil
}
