package models

import "time"

type Inventory struct {
	Refreshed          time.Time            `json:"refreshed"`
	SoftwareInventory  []SoftwareInventory  `json:"software_inventory"`
	ContainerInventory []ContainerInventory `json:"container_inventory"`
}

type SoftwareInventory struct {
	Package string `json:"package"`
	Version string `json:"version"`
	Summary string `json:"summary"`
}

type ContainerInventory struct {
	ContainerID   string `json:"container_id"`
	ContainerName string `json:"container_name"`
	Status        string `json:"status"`
	ImageID       string `json:"image_id"`
	ImageName     string `json:"image_name"`
}
