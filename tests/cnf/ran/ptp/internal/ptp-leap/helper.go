package ptpleap

import (
	"fmt"
	"regexp"
	"strings"
)

var announcementPattern = regexp.MustCompile(`\n(\d+\s+\d+\s+#\s\d+\s[a-zA-Z]+\s\d{4})\n\n`)
var leapLinePattern = regexp.MustCompile(`^\s*\d+\s+\d+\s+#`)

// GetLastAnnouncement returns the last leap event announcement from a leap-configmap Data.
func GetLastAnnouncement(leapConfigMapData string) (string, error) {
	if len(leapConfigMapData) == 0 {
		return leapConfigMapData, nil
	}

	announcementSlice := announcementPattern.FindStringSubmatch(leapConfigMapData)

	if len(announcementSlice) < 2 {
		return "", fmt.Errorf("error finding the last announcement")
	}

	return announcementSlice[1], nil
}

// RemoveLastLeapAnnouncement removes the last "leap announcement" line,
// i.e., the last line that looks like: "<seconds> <offset> # <date>".
func RemoveLastLeapAnnouncement(s string) string {
	lines := strings.Split(s, "\n")

	for i := len(lines) - 1; i >= 0; i-- {
		if leapLinePattern.MatchString(lines[i]) {
			lines = append(lines[:i], lines[i+1:]...)

			break
		}
	}

	return strings.Join(lines, "\n")
}
