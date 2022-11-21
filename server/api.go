package chserver

import (
	"net/http"

	"github.com/gorilla/websocket"
)

const (
	ErrCodeMissingRouteVar = "ERR_CODE_MISSING_ROUTE_VAR"
	ErrCodeInvalidRequest  = "ERR_CODE_INVALID_REQUEST"
	ErrCodeAlreadyExist    = "ERR_CODE_ALREADY_EXIST"
)

var apiUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}
