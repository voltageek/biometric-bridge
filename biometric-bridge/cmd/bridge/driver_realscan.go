//go:build realscan

package main

import (
	"biometric-bridge/internal/driver"
	"biometric-bridge/internal/driver/realscan"
)

func init() {
	registerDriverFactory("realscan", func(libPath string) (driver.Driver, error) {
		return realscan.New(libPath)
	})
}
