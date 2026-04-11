// Package config handles YAML configuration parsing and validation for the
// Biometric Bridge. It defines all configuration structs and validates required
// fields, refusing to start on invalid configuration (FR-018).
package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// BridgeConfig is the top-level configuration loaded from config.yaml.
type BridgeConfig struct {
	Bridge   BridgeSettings    `yaml:"bridge"`
	Devices  []DeviceConfig    `yaml:"devices"`
	Events   EventSettings     `yaml:"events"`
	Log      LogSettings       `yaml:"log"`
	Driver   string            `yaml:"driver"`
	GSDK     *GSDKSettings     `yaml:"gsdk,omitempty"`
	BS2      *BS2Settings      `yaml:"bs2,omitempty"`
	RealScan *RealScanSettings `yaml:"realscan,omitempty"`
}

// BridgeSettings holds HTTP server and authentication settings.
type BridgeSettings struct {
	Listen        string `yaml:"listen"`
	AllowedOrigin string `yaml:"allowed_origin"`
	PublicKeyFile string `yaml:"public_key_file"`
	TokenIssuer   string `yaml:"token_issuer"`
	TokenAudience string `yaml:"token_audience"`
	ClockSkew     string `yaml:"clock_skew"`
}

// DeviceConfig represents a single device as declared in config.yaml.
type DeviceConfig struct {
	Name   string `yaml:"name"`
	Addr   string `yaml:"addr"`
	Port   int    `yaml:"port"`
	UseSSL bool   `yaml:"use_ssl"`
}

// EventSettings holds reconnection backoff parameters.
type EventSettings struct {
	ReconnectBase string `yaml:"reconnect_base"`
	ReconnectCap  string `yaml:"reconnect_cap"`
}

// LogSettings holds logging configuration.
type LogSettings struct {
	Level string `yaml:"level"`
}

// GSDKSettings holds G-SDK driver configuration.
type GSDKSettings struct {
	GatewayAddr   string `yaml:"gateway_addr"`
	GatewayCACert string `yaml:"gateway_ca_cert"`
}

// BS2Settings holds BioStar 2 Device SDK configuration.
type BS2Settings struct {
	LibPath string `yaml:"lib_path"`
}

// RealScanSettings holds RealScan SDK configuration for USB fingerprint readers.
type RealScanSettings struct {
	LibPath string `yaml:"lib_path"`
}

// Load reads and parses a YAML config file, applies defaults, and validates
// all fields. It returns an error describing the first validation failure.
func Load(path string) (*BridgeConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg BridgeConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	applyDefaults(&cfg)

	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

// ClockSkewDuration parses the clock_skew field as a time.Duration.
func (b *BridgeSettings) ClockSkewDuration() time.Duration {
	d, _ := time.ParseDuration(b.ClockSkew)
	return d
}

// ReconnectBaseDuration parses reconnect_base as a time.Duration.
func (e *EventSettings) ReconnectBaseDuration() time.Duration {
	d, _ := time.ParseDuration(e.ReconnectBase)
	return d
}

// ReconnectCapDuration parses reconnect_cap as a time.Duration.
func (e *EventSettings) ReconnectCapDuration() time.Duration {
	d, _ := time.ParseDuration(e.ReconnectCap)
	return d
}

func applyDefaults(cfg *BridgeConfig) {
	if cfg.Bridge.Listen == "" {
		cfg.Bridge.Listen = "127.0.0.1:7070"
	}
	if cfg.Bridge.TokenAudience == "" {
		cfg.Bridge.TokenAudience = "biometric-bridge"
	}
	if cfg.Bridge.ClockSkew == "" {
		cfg.Bridge.ClockSkew = "30s"
	}
	if cfg.Events.ReconnectBase == "" {
		cfg.Events.ReconnectBase = "1s"
	}
	if cfg.Events.ReconnectCap == "" {
		cfg.Events.ReconnectCap = "120s"
	}
	if cfg.Log.Level == "" {
		cfg.Log.Level = "info"
	}
}

func validate(cfg *BridgeConfig) error {
	// Bridge settings
	if err := validateListenAddr(cfg.Bridge.Listen); err != nil {
		return fmt.Errorf("bridge.listen: %w", err)
	}
	if cfg.Bridge.AllowedOrigin == "" {
		return fmt.Errorf("bridge.allowed_origin is required")
	}
	if cfg.Bridge.PublicKeyFile == "" {
		return fmt.Errorf("bridge.public_key_file is required")
	}
	if cfg.Bridge.TokenIssuer == "" {
		return fmt.Errorf("bridge.token_issuer is required")
	}
	if _, err := time.ParseDuration(cfg.Bridge.ClockSkew); err != nil {
		return fmt.Errorf("bridge.clock_skew: invalid duration: %w", err)
	}

	// Driver
	switch cfg.Driver {
	case "bs2":
		if cfg.BS2 == nil {
			return fmt.Errorf("bs2 section is required when driver is \"bs2\"")
		}
		if cfg.BS2.LibPath == "" {
			return fmt.Errorf("bs2.lib_path is required")
		}
	case "gsdk":
		if cfg.GSDK == nil {
			return fmt.Errorf("gsdk section is required when driver is \"gsdk\"")
		}
		if cfg.GSDK.GatewayAddr == "" {
			return fmt.Errorf("gsdk.gateway_addr is required")
		}
	case "realscan":
		if cfg.RealScan == nil {
			return fmt.Errorf("realscan section is required when driver is \"realscan\"")
		}
		if cfg.RealScan.LibPath == "" {
			return fmt.Errorf("realscan.lib_path is required")
		}
	case "":
		return fmt.Errorf("driver is required (\"bs2\", \"gsdk\", or \"realscan\")")
	default:
		return fmt.Errorf("driver: unknown driver %q (expected \"bs2\", \"gsdk\", or \"realscan\")", cfg.Driver)
	}

	// Devices
	if len(cfg.Devices) == 0 {
		return fmt.Errorf("at least one device is required")
	}
	usbDriver := cfg.Driver == "realscan" // USB drivers don't need addr/port
	names := make(map[string]bool, len(cfg.Devices))
	for i, d := range cfg.Devices {
		prefix := fmt.Sprintf("devices[%d]", i)
		if d.Name == "" {
			return fmt.Errorf("%s.name is required", prefix)
		}
		if names[d.Name] {
			return fmt.Errorf("%s.name: duplicate device name %q", prefix, d.Name)
		}
		names[d.Name] = true
		if !usbDriver {
			if d.Addr == "" {
				return fmt.Errorf("%s.addr is required", prefix)
			}
			if d.Port < 1 || d.Port > 65535 {
				return fmt.Errorf("%s.port: must be 1–65535, got %d", prefix, d.Port)
			}
		}
	}

	// Events
	if _, err := time.ParseDuration(cfg.Events.ReconnectBase); err != nil {
		return fmt.Errorf("events.reconnect_base: invalid duration: %w", err)
	}
	if _, err := time.ParseDuration(cfg.Events.ReconnectCap); err != nil {
		return fmt.Errorf("events.reconnect_cap: invalid duration: %w", err)
	}

	// Log
	switch cfg.Log.Level {
	case "error", "info", "debug":
		// valid
	default:
		return fmt.Errorf("log.level: must be \"error\", \"info\", or \"debug\", got %q", cfg.Log.Level)
	}

	return nil
}

func validateListenAddr(addr string) error {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid host:port format: %w", err)
	}
	if host == "" {
		return fmt.Errorf("host is required (use 127.0.0.1 for localhost)")
	}
	if ip := net.ParseIP(host); ip == nil {
		return fmt.Errorf("invalid IP address %q", host)
	}
	if !strings.HasPrefix(host, "127.") {
		return fmt.Errorf("must bind to localhost (127.x.x.x), got %q", host)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("port must be 1–65535, got %q", portStr)
	}
	return nil
}
