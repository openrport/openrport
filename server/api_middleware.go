package chserver

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
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
func intIsMinute(m interface{}) (*time.Duration, error) {
	parseable := fmt.Sprintf("%v", m)
	dur, err := time.ParseDuration(parseable)
	if err != nil {
		parseable = fmt.Sprintf("%vm", m)
		dur, err := time.ParseDuration(parseable)
		if err != nil {
			return nil, errors.New("invalid type")
		}
		return &dur, nil
	}
	return &dur, nil
}

// ED TODO: move this to plus repo
func (al *APIListener) extendedPermissionsMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// ED TODO: enable this
			// this should do nothing for r.Method == "GET"
			if r.Method == "GETenable_this" {
				next.ServeHTTP(w, r)
				return
			}

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

			tr, cr := al.userService.GetEffectiveUserExtendedPermissions(currUser)
			if len(tr) > 0 {
				for _, TunnelsRestricted := range tr {
					if TunnelsRestricted == nil {
						continue
					}
					// cycle through the keys of the tunnel restriction map (e.g. "auto-close")
					for pName := range TunnelsRestricted {
						switch TunnelsRestricted[pName].(type) {
						case bool:
							//   "skip-idle-timeout": false // The user is not allowed to enable `skip-idle-timeout` on a tunnel (skipIdleTimeoutQueryParam)
							//*   "auth_allowed": true // The user is allowed to enable http basic auth for a tunnel
							//*   "http_proxy": true // The user is allowed to enable the http proxy

							restriction := TunnelsRestricted[pName].(bool)
							pValue, _ := strconv.ParseBool(r.FormValue(pName))
							if !restriction && pValue != restriction { // all false are to disallow
								msg1, msg2 := messageEnforceDisallow(restriction)
								al.jsonErrorResponseWithTitle(w, http.StatusUnauthorized, fmt.Sprintf("%s %v value%s", msg1, pName, msg2))
								return
							}
							break
						case string:
							//*   "host_header": ":*" // The user can only add a host header matching the regular expression.
							// like with true or false but if the param content matches the regular expression

							restriction := TunnelsRestricted[pName].(string)
							pValue := r.FormValue(pName)
							r, err := regexp.Compile(restriction)
							if err != nil {
								// ED TODO: need a validation function for the extended permissions regexes, on save
								al.Debugf("invalid restriction regular expression %q: %v", restriction, err)
							}

							if !r.Match([]byte(pValue)) {
								msg1, msg2 := messageEnforceDisallow(false)
								al.jsonErrorResponseWithTitle(w, http.StatusUnauthorized, fmt.Sprintf("%s %v value%s", msg1, pName, msg2))
								return
							}
							break
						case []interface{}: // [ "stuff", "like" "this" ]
							// Using an empty list or omitting an object will remove any restrictions.
							// For example, if allowed is not present, or if "allowed": [] then any command can be used.
							// If denied is missing or empty, the command is not validated against the deny patterns.
							//*   "local": ["20000","20001"] // The user can only create tunnels that would use port 2000 or 20001 on the rport server.
							//*   "remote": ["22","3389"] // The user can only create tunnels to the remote ports 22 or 3389.
							//*   "scheme": ["ssh","rdp"] // Scheme must be SSH or RDP
							//*   "protocol": ["tcp", "udp", "tcp-udp"] // Any protocols are allowed AKA only tunnels that matches at least one protocol can be created

							rl := TunnelsRestricted[pName].([]interface{})
							restrictionList := make([]string, len(rl)) // only strings are allowed
							for i, v := range rl {
								restrictionList[i] = fmt.Sprint(v)
							}

							pValue := r.FormValue(pName)
							al.Debugf("[]string parameter %v=%v restriction %v", pName, pValue, restrictionList)
							found := false
							for _, restriction := range restrictionList {
								if pValue == restriction {
									found = true
									break
								}
							}
							if !found && len(restrictionList) > 0 {
								al.jsonErrorResponseWithTitle(w, http.StatusUnauthorized, fmt.Sprintf("%v=%v must be one of %v", pName, pValue, restrictionList))
								return
							}
							break
						case map[string]interface{}: // { stuff like this } like this { "max" : "60m" }
							restriction := TunnelsRestricted[pName].(map[string]interface{})
							pValue := r.FormValue(pName)
							al.Debugf("map[string]interface{} parameter %v=%v restriction %v", pName, pValue, restriction)
							durPValue, err := intIsMinute(pValue)
							if err != nil { // ED TODO: what to do if the parsing of the parameter fails? 500?
								al.jsonErrorResponseWithTitle(w, http.StatusUnauthorized, fmt.Sprintf("parameter %v not parseable as time.duration", pName))
								return
							}
							if restriction["min"] != nil {
								//   "idle-timeout-minutes": { "min" : 5 } // On tunnel creation, the idle time out must be at least 5 minutes.
								min, err := intIsMinute(restriction["min"])
								if err != nil {
									al.jsonErrorResponseWithTitle(w, http.StatusUnauthorized, fmt.Sprintf("restriction %v not parseable as time.duration", restriction["min"]))
									return
								}
								if *durPValue <= *min {
									al.jsonErrorResponseWithTitle(w, http.StatusUnauthorized, fmt.Sprintf("%v=%v must be greater than %v", pName, pValue, min))
									return
								}
							}
							if restriction["max"] != nil { // ED TODO: this is not working, the max is not enforced
								//   "auto-close": { "max":  "60m" } // Auto-close must be used, with a maximum of 60m, that means the user will not be able to use the tunnel for more than 60 minutes.
								//	 If the user tries to create a tunnel without auto-close or with auto-close greater than 60m, it's forbidden. AKA this rule is about enforcing auto-close

								max, err := intIsMinute(restriction["max"])
								if err != nil {
									al.jsonErrorResponseWithTitle(w, http.StatusUnauthorized, fmt.Sprintf("restriction %v not parseable as time.duration", restriction["max"]))
									return
								}
								if *durPValue > *max {
									al.jsonErrorResponseWithTitle(w, http.StatusUnauthorized, fmt.Sprintf("%v=%v must be less than %v", pName, pValue, max))
									return
								}
							}
							break
						default:
							log.Printf(">>>>>>>>> : %v of type %T not recognized", TunnelsRestricted[pName], TunnelsRestricted[pName])
						}
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
