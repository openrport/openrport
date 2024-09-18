package inventory

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/openrport/openrport/share/logger"
	"github.com/openrport/openrport/share/models"
)

type DockerContainerInventoryManager struct{}

func NewDockerContainerInventoryManager() *DockerContainerInventoryManager {
	return &DockerContainerInventoryManager{}
}

func (p *DockerContainerInventoryManager) IsAvaliable(ctx context.Context) bool {
	_, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	return err == nil
}

func (p *DockerContainerInventoryManager) GetInventory(ctx context.Context, logger *logger.Logger) ([]models.ContainerInventory, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return nil, err
	}

	var result []models.ContainerInventory = make([]models.ContainerInventory, 0)
	for _, container := range containers {
		result = append(result, models.ContainerInventory{
			ContainerID:   container.ID,
			ContainerName: container.Names[0],
			Status:        container.Status,
			ImageID:       container.ImageID,
			ImageName:     container.Image,
		})
	}

	return result, nil
}
