package fundamentals

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestEndpointURLDoesNotExposeAPIKey(t *testing.T) {
	t.Parallel()

	client := NewClient(http.DefaultClient, "top-secret-key")
	got := client.endpointURL("/income-statement/AMZN?period=annual&limit=4")
	if strings.Contains(got, "top-secret-key") || strings.Contains(got, "apikey=") {
		t.Fatalf("endpointURL exposed the API key: %s", got)
	}
	if !strings.Contains(got, "period=annual") {
		t.Fatalf("endpointURL lost non-secret query values: %s", got)
	}
}

func TestProviderErrorDoesNotExposeAPIKey(t *testing.T) {
	t.Parallel()

	const secret = "top-secret-key"
	client := NewClient(&http.Client{Transport: fundamentalsRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Query().Get("apikey") != secret {
			t.Fatalf("request did not include configured API key")
		}
		return &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Status:     "429 Too Many Requests",
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{}`)),
		}, nil
	})}, secret)

	_, err := client.Snapshot(context.Background(), "AMZN")
	if err == nil {
		t.Fatal("Snapshot accepted a rate-limited profile response")
	}
	if strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), "apikey=") {
		t.Fatalf("provider error exposed API credentials: %v", err)
	}
}

func TestNetworkErrorDoesNotExposeAPIKeyInWrappedURL(t *testing.T) {
	t.Parallel()

	const secret = "top-secret-key"
	client := NewClient(&http.Client{Transport: fundamentalsRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return nil, errors.New("transport failed")
	})}, secret)

	_, err := client.Snapshot(context.Background(), "AMZN")
	if err == nil {
		t.Fatal("Snapshot accepted a network failure")
	}
	if strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), "apikey=") {
		t.Fatalf("network error exposed API credentials: %v", err)
	}
}

type fundamentalsRoundTripFunc func(*http.Request) (*http.Response, error)

func (fn fundamentalsRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
