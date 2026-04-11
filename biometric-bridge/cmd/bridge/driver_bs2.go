//go:build bs2

package main

import (
	"biometric-bridge/internal/driver"
	"biometric-bridge/internal/driver/bs2"
)

func init() {
	registerDriverFactory("bs2", func(libPath string) (driver.Driver, error) {
		return bs2.New(libPath)
	})
}
