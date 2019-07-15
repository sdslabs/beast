package utils

import (
	"time"
	"strconv"
	"fmt"
)

func GetDurationFromTimestamp(ts string) (time.Duration, error) {
	i, err := strconv.ParseInt(ts, 10, 64)
    if err != nil {
        return 0, fmt.Errorf("Timestamp provided is not valid: %s : %s", ts, err)
	}
	
	tm := time.Unix(i, 0)
	
	return tm.Sub(time.Now()), nil
}
