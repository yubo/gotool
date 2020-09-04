package util

import (
	"fmt"
	"time"
)

const (
	TIME_F_SIMPLE = "2006-01-02 15:04:05"
	TIME_F_STD    = "2006-01-02 15:04:05 -07"
	DATE_F_STD    = "2006-01-02"
	TIME_M        = int64(60)
	TIME_H        = int64(60 * 60)
	TIME_D        = int64(60 * 60 * 24)
)

func FmtTs(ts int64, verbose ...bool) string {
	if len(verbose) > 0 {
		return time.Unix(ts, 0).Format(TIME_F_STD)
	}
	return time.Unix(ts, 0).Format(TIME_F_SIMPLE)
}

func Until(f func(), interval time.Duration, stopCh <-chan struct{}) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// first call
		select {
		case <-stopCh:
			return
		default:
			f()
		}

		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				f()
			}
		}
	}()
}

func ParseDate(value *string, hour, min, sec int) (*int64, error) {
	var t time.Time
	var err error

	if value == nil {
		return nil, nil
	} else if t, err = time.Parse(DATE_F_STD, *value); err != nil {
		return nil, err
	}

	year, month, day := t.Date()
	return Int64(time.Date(year, month, day, hour, min, sec, 0, t.Location()).Unix()), nil
}

func FromNowAbs(s int64) string {
	if s < TIME_M {
		return fmt.Sprintf("%ds", s)
	}

	if s < TIME_H {
		return fmt.Sprintf("%dm", s/TIME_M)
	}

	if s < TIME_D {
		return fmt.Sprintf("%dh", s/TIME_H)
	}

	return fmt.Sprintf("%dd", s/TIME_D)
}

// FromNow return relative time with current time
func FromNow(s int64) string {
	if s == 0 {
		return "-"
	}

	if now := time.Now().Unix(); now > s {
		return FromNowAbs(now-s) + " ago"
	} else {
		return FromNowAbs(s-now) + " later"
	}
}

func Now() int64 {
	return time.Now().Unix()
}

func NowPtr(delta ...int64) *int64 {
	n := time.Now().Unix()
	for _, i := range delta {
		n += i
	}
	return &n
}

func TimeOf(v string) int64 {
	var n int64

	switch byte(v[len(v)-1]) {
	case 's', 'S':
		n = toInt64(v[:len(v)-1])
	case 'm', 'M':
		n = toInt64(v[:len(v)-1]) * 60
	case 'h', 'H':
		n = toInt64(v[:len(v)-1]) * 3600
	case 'd', 'D':
		n = toInt64(v[:len(v)-1]) * 3600 * 24
	default:
		n = toInt64(v)
	}
	return n
}
