package ai

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	co2Pattern     = regexp.MustCompile(`\$([0-9]+(?:\.[0-9]+)?)\$`)
	numberRegex    = regexp.MustCompile(`[0-9]+(?:\.[0-9]+)?`)
	unitRegex      = regexp.MustCompile(`^[ \t]*([a-zA-Z%/]+)`)
	ErrParseFailed = errors.New("parse_failed")
)

// ParseCO2 extracts the first numeric value (with optional $...$ envelope). It first tries the strict $<number>$ format,
// and falls back to the first number found in the text (e.g. "300 gCO2e").
func ParseCO2(text string) (float64, error) {
	val, _, err := ParseCO2WithUnit(text)
	return val, err
}

// ParseCO2WithUnit returns the parsed numeric value and unit (if any).
func ParseCO2WithUnit(text string) (float64, string, error) {
	// strict
	m := co2Pattern.FindStringSubmatch(text)
	if len(m) >= 2 {
		v, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			return 0, "", fmt.Errorf("%w: %v", ErrParseFailed, err)
		}
		return v, "", nil
	}
	// fallback: first number + optional unit
	matches := numberRegex.FindAllStringIndex(text, -1)
	if len(matches) == 0 {
		return 0, "", fmt.Errorf("%w: no co2 value found", ErrParseFailed)
	}
	bestIdx := matches[0]
	for _, m := range matches[1:] {
		if (m[1] - m[0]) > (bestIdx[1] - bestIdx[0]) {
			bestIdx = m
		}
	}
	valStr := text[bestIdx[0]:bestIdx[1]]
	v, err := strconv.ParseFloat(valStr, 64)
	if err != nil {
		return 0, "", fmt.Errorf("%w: %v", ErrParseFailed, err)
	}
	unit := ""
	if post := text[bestIdx[1]:]; post != "" {
		if u := unitRegex.FindStringSubmatch(post); len(u) >= 2 {
			unit = strings.TrimSpace(u[1])
		}
	}
	return v, unit, nil
}
