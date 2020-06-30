package chserver

import (
	chshare "github.com/cloudradar-monitoring/rport/share"
)

type ClientSession struct {
	ID      string            `json:"id"`
	Version string            `json:"version"`
	Address string            `json:"address"`
	Remotes []*chshare.Remote `json:"remotes"`
}
