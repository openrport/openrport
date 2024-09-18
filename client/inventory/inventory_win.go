//go:build windows
// +build windows

package inventory

var softwareInventoryManagers = []SoftwareInventoryManager{
	NewWindowsSoftwareInventoryManager(),
}

var containerInventoryManagers = []ContainerInventoryManager{}
