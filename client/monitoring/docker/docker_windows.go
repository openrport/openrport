//go:build windows
// +build windows

package docker

func ContainerNameByID(_ string) (string, error) {
	return "", ErrorNotImplementedForOS
}
