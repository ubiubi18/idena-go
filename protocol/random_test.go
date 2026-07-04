package protocol

import "testing"

func TestSecureIntnBounds(t *testing.T) {
	if got := secureIntn(0); got != 0 {
		t.Fatalf("secureIntn(0) = %d, want 0", got)
	}

	for i := 0; i < 100; i++ {
		got := secureIntn(7)
		if got < 0 || got >= 7 {
			t.Fatalf("secureIntn(7) = %d, want value in [0, 7)", got)
		}
	}
}

func TestSecureFloat32Bounds(t *testing.T) {
	for i := 0; i < 100; i++ {
		got := secureFloat32()
		if got < 0 || got >= 1 {
			t.Fatalf("secureFloat32() = %f, want value in [0, 1)", got)
		}
	}
}
