package main

import (
	"biometric-bridge/internal/driver"
	"biometric-bridge/internal/driver/demo"
)

// driverFactories holds driver constructor functions registered by build-tag-gated
// init() functions. Each factory takes a config and returns a Driver.
var driverFactories = map[string]DriverFactory{}

// DriverFactory creates a driver from configuration parameters.
type DriverFactory func(libPath string) (driver.Driver, error)

// registerDriverFactory is called from init() in build-tag-gated files.
func registerDriverFactory(name string, factory DriverFactory) {
	driverFactories[name] = factory
}

// getDriverFactory returns a registered factory by name, or nil if not found.
func getDriverFactory(name string) DriverFactory {
	return driverFactories[name]
}

func init() {
	registerDriverFactory("demo", func(_ string) (driver.Driver, error) {
		return demo.New(demo.DefaultDemoConfig())
	})
}
