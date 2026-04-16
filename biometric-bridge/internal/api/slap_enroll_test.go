package api

import (
	"testing"

	"biometric-bridge/internal/driver"
)

func TestExpectedFingerCount(t *testing.T) {
	cases := []struct {
		mode   driver.CaptureMode
		expect int
	}{
		{driver.CaptureRightFour, 4},
		{driver.CaptureLeftFour, 4},
		{driver.CaptureTwoThumbs, 2},
	}

	for _, c := range cases {
		got := expectedFingerCount(c.mode)
		if got != c.expect {
			t.Fatalf("expectedFingerCount(%s) = %d; want %d", c.mode, got, c.expect)
		}
	}
}

func TestValidateQuality(t *testing.T) {
	fingers := []driver.ScanResult{
		{Quality: 80, Finger: driver.FingerRightIndex},
		{Quality: 50, Finger: driver.FingerRightRing},
	}

	// minQuality=60 -> one failure
	failures := validateQuality(fingers, 60)
	if len(failures) != 1 {
		t.Fatalf("validateQuality expected 1 failure, got %d", len(failures))
	}

	// minQuality=0 -> no failures
	failures = validateQuality(fingers, 0)
	if len(failures) != 0 {
		t.Fatalf("validateQuality expected 0 failures when minQuality=0, got %d", len(failures))
	}
}

func TestValidateSlapEnrollRequest(t *testing.T) {
	okReq := &slapEnrollRequest{
		DeviceID: "Device1",
		UserID:   "user-1",
		UserName: "Alice",
		Mode:     string(driver.CaptureRightFour),
	}
	if err := validateSlapEnrollRequest(okReq); err != nil {
		t.Fatalf("validateSlapEnrollRequest failed for valid request: %v", err)
	}

	badReq := &slapEnrollRequest{}
	if err := validateSlapEnrollRequest(badReq); err == nil {
		t.Fatalf("validateSlapEnrollRequest should have failed for empty request")
	}
}
