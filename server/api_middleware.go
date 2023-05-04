package chserver

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"github.com/realvnc-labs/rport/server/api"
	errors2 "github.com/realvnc-labs/rport/server/api/errors"
	"github.com/realvnc-labs/rport/server/api/users"
	"github.com/realvnc-labs/rport/server/bearer"
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
			}

			next.ServeHTTP(w, r)
		})

	}
}
func messageEnforceDisallow(s bool) (string, string) {
	if !s {
		return "You are not allowed to set", ""
	}
	return "You must set", " to true"
}

// ED TODO: move this to plus repo
func (al *APIListener) extendedPermissionsMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// ED TODO: enable this
			// this should do nothing for r.Method == "GET"
			// if r.Method == "GET" {
			// 	next.ServeHTTP(w, r)
			// 	return
			// }

			al.Debugf("extendedPermissionsMiddleware: %v %v", r.Method, r.URL.Path)

			if al.insecureForTests {
				next.ServeHTTP(w, r)
				return
			}

			currUser, err := al.getUserModelForAuth(r.Context())
			if err != nil {
				al.jsonError(w, err)
				return
			}

			// ED TODO check if plus is enabled

			// tr and cr are the restricted tunnels and commands arrays
			tr, cr := al.userService.GetEffectiveUserExtendedPermissions(currUser)
			if len(tr) > 0 {
				for _, TunnelsRestricted := range tr {
					if TunnelsRestricted == nil {
						continue
					}
					// cycle through the keys of the tunnel restriction map (e.g. "auto-close")
					for parameter := range TunnelsRestricted {

						// if the key is in the request...
						// if r.FormValue(parameter) != "" {

						switch TunnelsRestricted[parameter].(type) {
						case bool:
							restriction := TunnelsRestricted[parameter].(bool)
							requestedParam, _ := strconv.ParseBool(r.FormValue(parameter))

							al.Debugf("bool parameter %v=%v must be %v", parameter, requestedParam, restriction)
							if requestedParam != TunnelsRestricted[parameter] {
								msg1, msg2 := messageEnforceDisallow(restriction)
								al.jsonErrorResponseWithTitle(w, http.StatusUnauthorized, fmt.Sprintf("%s %v value%s", msg1, parameter, msg2))
								return
							}
							//   "skip-idle-timeout": false // The user is not allowed to enable `skip-idle-timeout` on a tunnel (skipIdleTimeoutQueryParam)
							//*   "auth_allowed": true // The user is allowed to enable http basic auth for a tunnel
							//*   "http_proxy": true // The user is allowed to enable the http proxy

						case string:
							// restriction := TunnelsRestricted[parameter].(string)
							requestedParam, _ := strconv.ParseBool(r.FormValue(parameter))
							al.Debugf("string parameter %v=%v must match regular expression %v", parameter, r.FormValue(parameter), TunnelsRestricted[parameter])
							if TunnelsRestricted[parameter] == requestedParam {
								al.jsonErrorResponseWithTitle(w, http.StatusUnauthorized, fmt.Sprintf("You are not allowed to use this %v value.", parameter))
								return
							}
							//*   "host_header": ":*" // The user can only add a host header matching the regular expression.
							// like with true or false but if the param content matches the regular expression

						case map[string]interface{}:
							// log.Printf(">>>>>>>>> : %v of type %T not recognized", t[k], t[k])
							// { stuff like this }
							//   "idle-timeout-minutes": { "min" : 5 } // On tunnel creation, the idle time out must be at least 5 minutes.
							//   "auto-close": { "max":  "60m" } // Auto-close must be used, with a maximum of 60m, that means the user will not be able to use the tunnel for more than 60 minutes.
							//	 If the user tries to create a tunnel without auto-close or with auto-closer greater than 60m, it's forbidden. AKA this rule is about enforcing auto-close

							//	dur, err := time.ParseDuration(durationStr)

						case []interface{}:
							// log.Printf(">>>>>>>>> : %v of type %T not recognized", t[k], t[k])
							// [ stuff like this ]
							//*   "local": ["20000","20001"] // The user can only create tunnels that would use port 2000 or 20001 on the rport server.
							//*   "remote": ["22","3389"] // The user can only create tunnels to the remote ports 22 or 3389.
							//*   "scheme": ["ssh","rdp"] // Scheme must be SSH or RDP
							//*   "protocol": ["tcp", "udp", "tcp-udp"] // Any protocols are allowed AKA only tunnels that matches at least one protocol can be created

						default:
							log.Printf(">>>>>>>>> : %v of type %T not recognized", TunnelsRestricted[parameter], TunnelsRestricted[parameter])
						}

						// } else {
						// 	log.Printf(">>>>>>>>> : %v not found in request", parameter)
						// }

					}

				}
			}
			if len(cr) > 0 {

			}

			next.ServeHTTP(w, r)
		})
	}
}

// ED TODO: this will be moved in plus
func (al *APIListener) validateExtendedTunnelPermissions() {
	// PUT /api/v1/clients/1BB64205-67F4-40F2-A175-C9D6E9ED0A4D/tunnels?remote=80&scheme=other&acl=127.0.0.1&idle-timeout-minutes=5&protocol=tcp%2Budp HTTP/1.1
	// PUT /api/v1/clients/1BB64205-67F4-40F2-A175-C9D6E9ED0A4D/tunnels?remote=3393&scheme=rdp&local=20000&acl=127.0.0.0%2F24,255.255.255.255%2F8&auto-close=12h30m&idle-timeout-minutes=23&protocol=tcp
	// PUT /api/v1/clients/1BB64205-67F4-40F2-A175-C9D6E9ED0A4D/tunnels?remote=3393&scheme=rdp&local=20000
	// &acl=127.0.0.0%2F24,255.255.255.255%2F8
	// &auto-close=12h30m&idle-timeout-minutes=23&protocol=tcp HTTP/1.1

}

// param_name allowed yes/no
// a function that checks if param_name is in the query string and returns if it is allowed or not
// returns (param is present and true in query string) && (!extendedPermissions[param_name])
func (al *APIListener) validateExtendedPermissions(param_name string, param_value string) {

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
