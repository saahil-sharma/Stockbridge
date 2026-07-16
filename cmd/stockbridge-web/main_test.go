package main

import "testing"

func TestListenAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		explicit string
		port     string
		want     string
		wantErr  bool
	}{
		{name: "default", want: ":8080"},
		{name: "provider port", port: "10000", want: ":10000"},
		{name: "explicit local address", explicit: "127.0.0.1:9090", port: "10000", want: "127.0.0.1:9090"},
		{name: "invalid provider port", port: "not-a-port", wantErr: true},
		{name: "out of range provider port", port: "70000", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := listenAddress(tt.explicit, tt.port)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("listenAddress(%q, %q) returned no error", tt.explicit, tt.port)
				}
				return
			}
			if err != nil {
				t.Fatalf("listenAddress(%q, %q) returned error: %v", tt.explicit, tt.port, err)
			}
			if got != tt.want {
				t.Fatalf("listenAddress(%q, %q) = %q, want %q", tt.explicit, tt.port, got, tt.want)
			}
		})
	}
}

func TestDisplayURL(t *testing.T) {
	t.Parallel()

	if got := displayURL(":8080"); got != "http://localhost:8080" {
		t.Fatalf("displayURL(:8080) = %q", got)
	}
	if got := displayURL("127.0.0.1:9090"); got != "http://127.0.0.1:9090" {
		t.Fatalf("displayURL(127.0.0.1:9090) = %q", got)
	}
}
