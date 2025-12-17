package ai

import (
	"errors"
	"regexp"
	"strconv"
)

var co2Pattern = regexp.MustCompile(`\$([0-9]+(?:\.[0-9]+)?)\$`)

// ParseCO2 extracts $<number>$ and returns the parsed float64.
func ParseCO2(text string) (float64, error) {
	m := co2Pattern.FindStringSubmatch(text)
	if len(m) < 2 {
		return 0, errors.New("no co2 value found")
	}
	v, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, err
	}
	return v, nil
}
