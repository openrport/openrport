package chserver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	rportplus "github.com/realvnc-labs/rport/plus"
	"github.com/realvnc-labs/rport/server/api"
	errors2 "github.com/realvnc-labs/rport/server/api/errors"
	"github.com/realvnc-labs/rport/server/api/users"
	"github.com/realvnc-labs/rport/server/bearer"
	"github.com/realvnc-labs/rport/server/clients/clienttunnel"
	"github.com/realvnc-labs/rport/server/routes"
	"github.com/realvnc-labs/rport/share/enums"
	"github.com/realvnc-labs/rport/share/logger"
)

func (al *APIListener) wrapStaticPassModeMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if al.userService.GetProviderType() == enums.ProviderSourceStatic {
			al.jsonError(w, errors2.APIError{
				HTTPStatus: http.StatusBadRequest,
				Message:    "server runs on a static user-password pair, please use JSON file or database for user data",
			})
			return
		}
		next.ServeHTTP(w, r)
	}
}

func (al *APIListener) wrapAdminAccessMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if al.insecureForTests {
			next.ServeHTTP(w, r)
			return
		}

		user, err := al.getUserModelForAuth(r.Context())
		if err != nil {
			al.jsonError(w, err)
			return
		}

		if user.IsAdmin() {
			next.ServeHTTP(w, r)
			return
		}

		al.jsonError(w, errors2.APIError{
			Message: fmt.Sprintf(
				"current user should belong to %s group to access this resource",
				users.Administrators,
			),
			HTTPStatus: http.StatusForbidden,
		})
	})
}

func (al *APIListener) wrapTotPEnabledMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !al.config.API.TotPEnabled {
			al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, "TotP is disabled")
			return
		}

		next.ServeHTTP(w, r)
	}
}

func (al *APIListener) wrapWithAuthMiddleware(isBearerOnly bool) mux.MiddlewareFunc {
	return func(f http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authorized, username, err := al.lookupUser(r, isBearerOnly)
			if err != nil {
				al.Logf(logger.LogLevelError, err.Error())
				if errors.Is(err, ErrTooManyRequests) {
					al.jsonErrorResponse(w, http.StatusTooManyRequests, err)
					return
				}
				al.jsonErrorResponse(w, http.StatusInternalServerError, err)
				return
			}

			if !al.handleBannedIPs(r, authorized) {
				return
			}

			if !authorized || username == "" {
				al.bannedUsers.Add(username)
				al.jsonErrorResponse(w, http.StatusUnauthorized, errors.New("unauthorized"))
				return
			}

			newCtx := api.WithUser(r.Context(), username)

			token, hasBearerToken := bearer.GetBearerToken(r)
			if hasBearerToken {
				err = al.updateTokenAccess(newCtx, token, time.Now(), r.UserAgent(), r.RemoteAddr)
				if err != nil {
					al.jsonError(w, err)
					return
				}
			}

			f.ServeHTTP(w, r.WithContext(newCtx))
		})
	}
}

func (al *APIListener) wrapClientAccessMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if al.insecureForTests {
			next.ServeHTTP(w, r)
			return
		}

		vars := mux.Vars(r)
		clientID := vars[routes.ParamClientID]
		if clientID == "" {
			al.jsonErrorResponseWithTitle(w, http.StatusBadRequest, fmt.Sprintf("Missing %q route param.", routes.ParamClientID))
			return
		}

		curUser, err := al.getUserModelForAuth(r.Context())
		if err != nil {
			al.jsonError(w, err)
			return
		}

		clientGroups, err := al.clientGroupProvider.GetAll(r.Context())
		if err != nil {
			al.jsonError(w, err)
		}
		err = al.clientService.CheckClientAccess(clientID, curUser, clientGroups)
		if err != nil {
			al.jsonError(w, err)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (al *APIListener) extendedPermissionDeleteTunnelRaw(tunnel *clienttunnel.Tunnel, currUser *users.User) error {
	// TODO: this should be moved in the permission middleware
	if rportplus.IsPlusEnabled(al.config.PlusConfig) && !currUser.IsAdmin() && tunnel.Owner != currUser.Username {
		plusPermissionCapability := al.Server.plusManager.GetExtendedPermissionCapabilityEx()
		tr, _ := al.userService.GetEffectiveUserExtendedPermissions(currUser)
		if tr != nil {
			err := plusPermissionCapability.ValidateExtendedDeleteNonOwnedTunnelPermissionRaw(tr)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (al *APIListener) extendedPermissionCommandRaw(cmd string, currUser *users.User) error {
	if rportplus.IsPlusEnabled(al.config.PlusConfig) {
		// check only if plus is enabled, no error otherwise
		plusPermissionCapability := al.Server.plusManager.GetExtendedPermissionCapabilityEx()
		_, cr := al.userService.GetEffectiveUserExtendedPermissions(currUser)
		if cr != nil {
			err := plusPermissionCapability.ValidateExtendedCommandPermissionRaw(cmd, false, cr)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (al *APIListener) permissionsMiddleware(permission string) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if al.insecureForTests {
				next.ServeHTTP(w, r)
				return
			}

			currUser, err := al.getUserModelForAuth(r.Context())
			if err != nil {
				al.jsonError(w, err)
				return
			}

			if al.userService.SupportsGroupPermissions() {
				// Check group permissions only if supported otherwise let pass.
				err = al.userService.CheckPermission(currUser, permission)
				if err != nil {
					al.jsonError(w, err)
					return
				}
				if rportplus.IsPlusEnabled(al.config.PlusConfig) &&
					(permission == users.PermissionTunnels ||
						permission == users.PermissionCommands ||
						permission == users.PermissionScheduler) {
					plusPermissionCapability := al.Server.plusManager.GetExtendedPermissionCapabilityEx()
					al.Debugf("extended \"%s\" permission middleware: %v %v", permission, r.Method, r.URL.Path)
					tr, cr := al.userService.GetEffectiveUserExtendedPermissions(currUser)
					switch permission {
					case users.PermissionTunnels:
						if tr != nil {
							err = plusPermissionCapability.ValidateExtendedTunnelPermission(r, tr)
						}
					case users.PermissionCommands, users.PermissionScheduler:
						if cr != nil {
							err = plusPermissionCapability.ValidateExtendedCommandPermission(r, cr)
						}
					}
					if err != nil {
						al.jsonErrorResponseWithDetail(w, http.StatusBadRequest, "", err.Error(), "")
						return
					}
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (al *APIListener) updateTokenAccess(ctx context.Context, token string, accessTime time.Time, userAgent string, remoteAddress string) (err error) {
	tokenCtx, err := bearer.ParseToken(token, al.config.API.JWTSecret)
	if err != nil {
		al.Debugf("failed to parse jwt token: %v", err)
		return err
	}

	// at least make sure the source jwt was valid. not quite sure why ParseToken doesn't do this.
	if !tokenCtx.JwtToken.Valid {
		err := errors.New("jwt token is invalid")
		al.Debugf("%v", err)
		return err
	}

	found, sessionInfo, err := al.apiSessions.Get(ctx, tokenCtx.AppClaims.SessionID)
	if err != nil {
		return err
	}

	// if no session cache yet, then don't try to update
	if !found {
		return nil
	}

	sessionInfo.LastAccessAt = accessTime
	sessionInfo.UserAgent = userAgent
	sessionInfo.IPAddress = remoteAddress

	_, err = al.apiSessions.Save(ctx, sessionInfo)
	if err != nil {
		return err
	}

	return nil
}
