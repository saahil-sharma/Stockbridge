package symbols

import (
	"fmt"
	"regexp"
	"strings"
)

var tickerPattern = regexp.MustCompile(`^[A-Z][A-Z0-9.-]{0,13}$`)

func NormalizeTicker(input string) (string, error) {
	ticker := strings.ToUpper(strings.TrimSpace(input))
	if ticker == "" {
		return "", fmt.Errorf("ticker is required")
	}
	if !tickerPattern.MatchString(ticker) {
		return "", fmt.Errorf("invalid ticker %q: use 1-14 letters, digits, dots, or dashes", input)
	}
	return ticker, nil
}
