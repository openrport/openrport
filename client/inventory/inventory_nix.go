//go:build !windows
// +build !windows

package inventory

var softwareInventoryManagers = []SoftwareInventoryManager{
	NewDPKGSoftwareInventoryManager(),
}

var containerInventoryManagers = []ContainerInventoryManager{
	NewDockerContainerInventoryManager(),
}
