package demo

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
)

// generateRandomBytes returns n random bytes using crypto/rand.
func generateRandomBytes(n int) []byte {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return b
}

// generateRandomInt returns a random int in [min, max].
func generateRandomInt(min, max int) int {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(max-min+1)))
	return min + int(n.Int64())
}

// GenerateTemplate returns a base64-encoded random template suitable for a scan result.
func GenerateTemplate(size int) string {
	return base64.StdEncoding.EncodeToString(generateRandomBytes(size))
}

// GenerateScanResult creates a synthetic ScanResult for the given device config.
func GenerateScanResult(dc MockDeviceConfig, qualityMin, qualityMax int) (template string, quality int, width, height int) {
	return GenerateTemplate(2048),
		generateRandomInt(qualityMin, qualityMax),
		dc.ScanWidth,
		dc.ScanHeight
}

// FindDevice returns the MockDeviceConfig for the given name, or nil if not found.
func FindDevice(cfg DemoConfig, name string) *MockDeviceConfig {
	for i := range cfg.Devices {
		if cfg.Devices[i].Name == name {
			return &cfg.Devices[i]
		}
	}
	return nil
}

// ValidateDevice returns an error if the device name is not known.
func ValidateDevice(cfg DemoConfig, name string) error {
	if FindDevice(cfg, name) == nil {
		return fmt.Errorf("device not found: %s", name)
	}
	return nil
}

// SlapFingerNames returns the finger names for a given CaptureMode.
func SlapFingerNames(mode string) []string {
	switch mode {
	case "left_four":
		return []string{"left_little", "left_ring", "left_middle", "left_index"}
	case "right_four":
		return []string{"right_index", "right_middle", "right_ring", "right_little"}
	case "two_thumbs":
		return []string{"right_thumb", "left_thumb"}
	default:
		return nil
	}
}

// GenerateSlapResult creates a synthetic SlapScanResult for the given device config and mode.
func GenerateSlapResult(dc MockDeviceConfig, mode string, qualityMin, qualityMax int) (slapImage string, slapWidth, slapHeight int, fingers []map[string]any) {
	slapWidth = dc.SlapWidth
	slapHeight = dc.SlapHeight
	slapImage = GenerateTemplate(slapWidth * slapHeight)

	fingerNames := SlapFingerNames(mode)
	fingers = make([]map[string]any, 0, len(fingerNames))
	for _, name := range fingerNames {
		fingers = append(fingers, map[string]any{
			"finger":  name,
			"image":   GenerateTemplate(2048),
			"width":   dc.ScanWidth,
			"height":  dc.ScanHeight,
			"quality": generateRandomInt(qualityMin, qualityMax),
		})
	}

	return
}
