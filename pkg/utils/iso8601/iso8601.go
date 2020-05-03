package iso8601

import "time"

const Format = "2006-01-02T15:04:05.000Z"

var fixedZone = time.FixedZone("", 0)

func ToStringZmillis(pTime *time.Time) string {
	if pTime == nil {
		now := time.Now()
		pTime = &now
	}
	return pTime.Format(Format)
}

func FromStringZmillis(pTime string) (time.Time, error) {
	return time.ParseInLocation(Format, pTime, fixedZone)
}
