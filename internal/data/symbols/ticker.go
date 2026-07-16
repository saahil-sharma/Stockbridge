package symbols

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var tickerPattern = regexp.MustCompile(`^[A-Z][A-Z0-9.-]{0,13}$`)

var ErrInvalidTicker = errors.New("invalid ticker")

func NormalizeTicker(input string) (string, error) {
	ticker := strings.ToUpper(strings.TrimSpace(input))
	if ticker == "" {
		return "", fmt.Errorf("%w: ticker is required", ErrInvalidTicker)
	}
	if !tickerPattern.MatchString(ticker) {
		return "", fmt.Errorf("%w %q: use 1-14 letters, digits, dots, or dashes", ErrInvalidTicker, input)
	}
	return ticker, nil
}

func EquivalentTickers(left, right string) bool {
	left = canonicalTicker(left)
	right = canonicalTicker(right)
	return left != "" && left == right
}

func TickerVariants(input string) []string {
	normalized, err := NormalizeTicker(input)
	if err != nil {
		return nil
	}
	variants := []string{normalized}
	if strings.Contains(normalized, ".") {
		variants = append(variants, strings.ReplaceAll(normalized, ".", "-"))
	}
	if strings.Contains(normalized, "-") {
		variants = append(variants, strings.ReplaceAll(normalized, "-", "."))
	}
	return uniqueStrings(variants)
}

func canonicalTicker(input string) string {
	ticker := strings.ToUpper(strings.TrimSpace(input))
	ticker = strings.ReplaceAll(ticker, "-", ".")
	return ticker
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	unique := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	return unique
}
