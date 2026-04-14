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

func GenerateSlapResult(dc MockDeviceConfig, mode string, qualityMin, qualityMax int) (slapImage string, slapWidth, slapHeight int, fingers []map[string]any) {
	slapWidth = dc.SlapWidth
	slapHeight = dc.SlapHeight
	slapImage = base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0xCD}, slapWidth*slapHeight))

	fingerNames := SlapFingerNames(mode)
	fingers = make([]map[string]any, 0, len(fingerNames))
	for _, name := range fingerNames {
		fingers = append(fingers, map[string]any{
			"finger":  name,
			"image":   stubTemplateB64(),
			"width":   dc.ScanWidth,
			"height":  dc.ScanHeight,
			"quality": stubQuality,
		})
	}

	return
}
