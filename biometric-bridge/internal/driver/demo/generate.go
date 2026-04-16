package demo

import (
	"bytes"
	"encoding/base64"
	"fmt"
)

const stubTemplateSize = 2048

var stubTemplate []byte

func init() {
	stubTemplate = bytes.Repeat([]byte{0xAB}, stubTemplateSize)
}

func stubTemplateB64() string {
	return base64.StdEncoding.EncodeToString(stubTemplate)
}

const stubQuality = 85

// GenerateScanResult returns a single-finger scan template and quality.
// This remains deterministic (stubQuality) for now so existing demo behavior
// for single-finger scans is preserved.
func GenerateScanResult(dc MockDeviceConfig, qualityMin, qualityMax int) (template string, quality int, width, height int) {
	return stubTemplateB64(), stubQuality, dc.ScanWidth, dc.ScanHeight
}

func FindDevice(cfg DemoConfig, name string) *MockDeviceConfig {
	for i := range cfg.Devices {
		if cfg.Devices[i].Name == name {
			return &cfg.Devices[i]
		}
	}
	return nil
}

func ValidateDevice(cfg DemoConfig, name string) error {
	if FindDevice(cfg, name) == nil {
		return fmt.Errorf("device not found: %s", name)
	}
	return nil
}

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

// GenerateSlapResult creates a synthetic slap image and per-finger data.
// It accepts an attempt counter and demo config so the demo driver can
// simulate improving quality across successive SlapScan attempts.
func GenerateSlapResult(dc MockDeviceConfig, mode string, qualityMin, qualityMax int, attempt int, cfg DemoConfig) (slapImage string, slapWidth, slapHeight int, fingers []map[string]any) {
	slapWidth = dc.SlapWidth
	slapHeight = dc.SlapHeight
	slapImage = base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0xCD}, slapWidth*slapHeight))

	fingerNames := SlapFingerNames(mode)
	fingers = make([]map[string]any, 0, len(fingerNames))
	// Determine the quality for this attempt. If the demo config specifies
	// InitialQuality/ImprovedQuality and AttemptsToImprove, use those values
	// to simulate improvement after a given number of attempts. Fall back to
	// stubQuality if no improvement config is provided.
	quality := stubQuality
	if cfg.AttemptsToImprove > 0 {
		// initialize fallback values
		initial := cfg.InitialQuality
		improved := cfg.ImprovedQuality
		if initial == 0 {
			initial = qualityMin
		}
		if improved == 0 {
			improved = qualityMax
		}
		if attempt <= cfg.AttemptsToImprove {
			quality = initial
		} else {
			quality = improved
		}
	}
	// Clamp quality to the supplied min/max range
	if quality < qualityMin {
		quality = qualityMin
	}
	if quality > qualityMax {
		quality = qualityMax
	}

	for _, name := range fingerNames {
		fingers = append(fingers, map[string]any{
			"finger":  name,
			"image":   stubTemplateB64(),
			"width":   dc.ScanWidth,
			"height":  dc.ScanHeight,
			"quality": quality,
		})
	}

	return
}
