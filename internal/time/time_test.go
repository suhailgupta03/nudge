package time

import (
	"bytes"
	"os"
	"testing"
	"time"
)

func TestNudgeTime(t *testing.T) {
	var buf bytes.Buffer

	tests := []struct {
		name        string
		timezoneVar string
	}{
		{"DefaultTimeZone", ""},
		{"CustomTimeZone", "America/New_York"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			os.Setenv("TZ", test.timezoneVar)

			nt := &NudgeTime{}
			result := nt.NudgeTime()

			if result == nil {
				t.Fatalf("Expected non-nil result, got nil")
			}
			loc, _ := time.LoadLocation("Local")
			expected := time.Now().In(loc)
			diff := result.Sub(expected)

			if diff > 1*time.Second || diff < -1*time.Second {
				t.Errorf("Expected time close to %v, got %v", expected, *result)
			}

			// Reset the buffer for the next test iteration
			buf.Reset()
		})
	}
}
