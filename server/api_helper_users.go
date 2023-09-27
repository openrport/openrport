package chserver

import (
	"context"
	"net/http"

	"github.com/openrport/openrport/server/api"
	errors2 "github.com/openrport/openrport/server/api/errors"
	"github.com/openrport/openrport/server/api/users"
)

// TODO: remove
func (al *APIListener) getUserModel(ctx context.Context) (*users.User, error) {
	curUsername := api.GetUser(ctx, al.Logger)
	if curUsername == "" {
		return nil, nil
	}

	user, err := al.userService.GetByUsername(curUsername)
	if err != nil {
		return nil, err
	}

	return user, err
}

// TODO: move to userService
func (al *APIListener) getUserModelForAuth(ctx context.Context) (*users.User, error) {
	usr, err := al.getUserModel(ctx)
	if err != nil {
		return nil, errors2.APIError{
			Err:        err,
			HTTPStatus: http.StatusInternalServerError,
		}
	}

	if usr == nil {
		return nil, errors2.APIError{
			Message:    "unauthorized access",
			HTTPStatus: http.StatusUnauthorized,
		}
	}

	return usr, nil
}
