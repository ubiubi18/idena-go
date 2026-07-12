package protocol

import "testing"

func TestShouldLogHandshakeFailure(t *testing.T) {
	tests := []struct {
		name           string
		currentVersion string
		peerVersion    string
		want           bool
	}{
		{name: "invalid current build label", currentVersion: "modern-f6db89a", peerVersion: "1.1.2", want: true},
		{name: "invalid peer version", currentVersion: "1.1.2", peerVersion: "development", want: true},
		{name: "newer major", currentVersion: "1.1.2", peerVersion: "2.0.0", want: true},
		{name: "same minor", currentVersion: "1.1.2", peerVersion: "1.1.0", want: true},
		{name: "newer minor", currentVersion: "1.1.2", peerVersion: "1.2.0", want: true},
		{name: "older minor", currentVersion: "1.1.2", peerVersion: "1.0.9", want: false},
		{name: "older major", currentVersion: "2.0.0", peerVersion: "1.9.9", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldLogHandshakeFailure(tt.currentVersion, tt.peerVersion); got != tt.want {
				t.Fatalf("shouldLogHandshakeFailure(%q, %q) = %v, want %v", tt.currentVersion, tt.peerVersion, got, tt.want)
			}
		})
	}
}
