package utils

import "time"

// DurationSecondsCeil converts duration to seconds and rounds up. DurationSecondsCeil 将时长转换为秒并向上取整。
func DurationSecondsCeil(duration time.Duration) int64 {
	if duration <= 0 {
		return 0
	}
	seconds := int64(duration / time.Second)
	if duration%time.Second != 0 {
		seconds++
	}
	if seconds <= 0 {
		return 1
	}
	return seconds
}
