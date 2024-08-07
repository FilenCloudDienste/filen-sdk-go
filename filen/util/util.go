package util

import "time"

func TimestampToTime(timestamp int) time.Time {
	//TODO tmp
	return time.Unix(int64(timestamp), 0)
}
