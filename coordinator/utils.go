// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package coordinator

import (
	"time"
)

// hasPassed reports whether the provided duration
// added with the given time is before the current time or not.
func hasPassed(t time.Time, d time.Duration) bool {
	return time.Now().After(t.Add(d))
}

// min finds the minimum between the provided int values.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
