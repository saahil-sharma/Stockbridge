package market

import (
	"strings"
	"testing"
)

func TestParseYahooChart(t *testing.T) {
	t.Parallel()

	input := strings.NewReader(`{
		"chart": {
			"result": [{
				"timestamp": [1714478400, 1714564800, 1714651200],
				"indicators": {
					"quote": [{
						"close": [180.1, null, 185.3]
					}]
				}
			}],
			"error": null
		}
	}`)

	points, err := ParseYahooChart(input)
	if err != nil {
		t.Fatalf("ParseYahooChart returned error: %v", err)
	}
	if len(points) != 2 {
		t.Fatalf("len(points) = %d, want 2", len(points))
	}
	if points[0].Close != 180.1 || points[1].Close != 185.3 {
		t.Fatalf("unexpected closes: %#v", points)
	}
}

func TestYahooTicker(t *testing.T) {
	t.Parallel()

	if got := yahooTicker("brk.b"); got != "BRK-B" {
		t.Fatalf("yahooTicker = %q, want BRK-B", got)
	}
}
