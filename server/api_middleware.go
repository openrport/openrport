package chserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	rportplus "github.com/realvnc-labs/rport/plus"
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
				if permission == users.PermissionTunnels || permission == users.PermissionCommands || permission == users.PermissionScheduler {
					if !rportplus.IsPlusEnabled(al.config.PlusConfig) { // that checks whether the plugin is enabled in the config -- if it is enabled but fails to load, then there will be an error and the server won't start
						al.jsonErrorResponseWithTitle(w, http.StatusForbidden, "Extended permissions validation failed because rport-plus plugin not loaded")
						return
					}
					al.Debugf("extended \"%s\" PermissionsMiddleware: %v %v", permission, r.Method, r.URL.Path)
					tr, cr := al.userService.GetEffectiveUserExtendedPermissions(currUser)
					switch permission {
					case users.PermissionTunnels:
						if tr != nil {
							err = validateExtendedTunnelPermission(r, tr)
						}
						break
					case users.PermissionCommands:
					case users.PermissionScheduler:
						if cr != nil {
							err = validateExtendedCommandPermission(r, cr)
						}
						break
					}
					if err != nil {
						al.jsonError(w, err)
						return
					}
				}
			}

			al.Debugf("ENDF PermissionsMiddleware: %v %v", r.Method, r.URL.Path)
			next.ServeHTTP(w, r)
		})

	}
}

// ED TODO: this INSIDE PLUS!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
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

// ED TODO: this INSIDE PLUS!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
func messageEnforceDisallow(s bool) (string, string) {
	if !s {
		return "You are not allowed to set", ""
	}
	return "You must set", " to true"
}

// ED TODO: this INSIDE PLUS!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
func errorMessageMaxMinLimits(pName string, pValue string, limit string, ruleValue string) string {
	mm := "greater"
	if ruleValue == "max" {
		mm = "less"
	}
	return fmt.Sprintf("4 Tunnel with %v=%v is forbidden. Allowed value must be %s than %v", pName, pValue, mm, limit)
}

// ED TODO: this INSIDE PLUS!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
// ED TODO: use shortDur in all messages that use time.Duration
func shortDur(d time.Duration) string {
	return fmt.Sprintf("%vm", d.Minutes())
}

// ED TODO: this INSIDE PLUS!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
func validateExtendedTunnelPermission(r *http.Request, tr []users.StringInterfaceMap) error {
	//  ED TODO: Plus plugin LICENSE CHECK The method will validate the permissions 5 times and then all validations will be denied with a message, "You are running the plus-plugin without a licence. Max 5 validation reached. Restart rportd to continue testing."
	if len(tr) > 0 && r.Method != "GET" {
		for _, TunnelsRestricted := range tr {
			// cycle through the keys of the tunnel restriction map (e.g. "auto-close")
			for pName := range TunnelsRestricted {
				switch TunnelsRestricted[pName].(type) {
				case bool:
					// given a bool param,
					//		if the restriction is false then the param can't be set (or it can be set only false);
					//		if the restriction is true (or there is no restriction for the param) then the param can be set (true or false)
					restriction := TunnelsRestricted[pName].(bool)
					pValue, _ := strconv.ParseBool(r.FormValue(pName))
					if !restriction && pValue != restriction { // all false are to disallow
						msg1, msg2 := messageEnforceDisallow(restriction)
						return errors.New(fmt.Sprintf("1 Tunnel with %v=%v is forbidden. %s %v value%s", pName, pValue, msg1, pName, msg2))
					}
					break
				case string: // like with true or false but if the param content matches the regular expression
					fmt.Printf("TunnelsRestricted[pName].(string) = %v\n", TunnelsRestricted[pName].(string))
					restriction := TunnelsRestricted[pName].(string)
					pValue := r.FormValue(pName)
					r, err := regexp.Compile(restriction)
					if err != nil {
						fmt.Printf("invalid restriction regular expression %q: %v", restriction, err) // ED TODO: need a validation function for the extended permissions regexes, on save
					}
					if !r.Match([]byte(pValue)) {
						return errors.New(fmt.Sprintf("2 Tunnel with %v=%v is forbidden. Allowed values must match '%v' regular expression", pName, pValue, restriction))
					}
					break //
				case []interface{}: // [ "stuff", "like" "this" ]
					pValue := r.FormValue(pName)
					if inList, _ := restrictionInList(pName, pValue, TunnelsRestricted,
						func(pValue string, restriction string) bool {
							return pValue == restriction
						}); !inList {
						paramStr := fmt.Sprintf("with parameter %s=%s", pName, pValue)
						if (pValue == "") || (pValue == "0") {
							paramStr = fmt.Sprintf("without parameter %s", pName)
						}
						return errors.New(fmt.Sprintf("3 Tunnel %s is forbidden. Allowed values: %v", paramStr, TunnelsRestricted[pName]))
					}
					break
				case map[string]interface{}: // stuff like this { "max": "60m", "min": "5m" }
					//	If the user tries to create a tunnel without auto-close or with auto-close greater than 60m, it's forbidden.
					// 	AKA this rule is about enforcing auto-close (min) and limiting it (max).
					restriction := TunnelsRestricted[pName].(map[string]interface{})
					pValue := r.FormValue(pName)
					for rule := range restriction {
						if pValue == "" {
							pValue = "0m"
						}
						// al.Debugf("map[string]interface{} rule(%v) parameter %v=%v restriction %v", rule, pName, pValue, restriction[rule])
						durPValue, err := intIsMinute(pValue)
						if err != nil { // ED TODO: what to do if the parsing of the parameter fails? 500?
							return errors.New(fmt.Sprintf("parameter %v not parseable as time.duration", pName))

						}
						ruleValue, err := intIsMinute(restriction[rule])
						if err != nil { // ED TODO: this should not happen, the validation should be done on save
							return errors.New(fmt.Sprintf("restriction %v not parseable as time.duration: %v", restriction[rule], err))
						}
						if (rule == "min" && *durPValue < *ruleValue) || (rule == "max" && *durPValue > *ruleValue) {
							return errors.New(errorMessageMaxMinLimits(pName, fmt.Sprintf("%v", durPValue.Minutes()), shortDur(*ruleValue), rule))
						}
					}
					break
				default:
					// ED TODO: TunnelsRestricted validation is a simple cycle like this, a function that cycle the type and checks if type IN [ list of admitted types for TunnelsRestricted])
					// al.Debugf("extended \"tunnels\" Permissions %v of type %T not recognized", TunnelsRestricted[pName], TunnelsRestricted[pName])
				}
			}
		}
	}
	return nil
}

func restrictionInList(pName string, pValue string, TunnelsRestricted users.StringInterfaceMap, restrictionMatch func(string, string) bool) (bool, string) {
	rl := TunnelsRestricted[pName].([]interface{})
	restrictionList := make([]string, len(rl)) // only strings are allowed
	for i, v := range rl {
		restrictionList[i] = fmt.Sprint(v)
	}
	found := false
	rFound := ""
	for _, restriction := range restrictionList {
		if restrictionMatch(pValue, restriction) {
			found = true
			rFound = restriction
			break
		}
	}
	return (found && len(restrictionList) > 0) || len(restrictionList) == 0, rFound
}

// ED TODO: this INSIDE PLUS!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
func validateExtendedCommandPermission(r *http.Request, cr []users.StringInterfaceMap) error {
	// check if we have a "cmd" or "command" parameter
	if r.Method != "GET" {
		isSudo := false
		if r.FormValue("is_sudo") != "" {
			isSudo, _ = strconv.ParseBool(r.FormValue("is_sudo"))
		}
		command := r.FormValue("cmd") // some endpoint use "cmd" instead of "command"
		if command == "" {
			command = r.FormValue("command")
		}
		if command == "" {
			bodyBytes, _ := ioutil.ReadAll(r.Body)                // read all request body
			r.Body.Close()                                        //  must close
			r.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes)) // creates a new one for the next handler

			var data map[string]interface{}
			err := json.Unmarshal(bodyBytes, &data) // sample data {"command": "/bin/date;foo;whoami", "timeout_sec": 0}
			if err == nil {
				if cmdValue, ok := data["command"].(string); ok {
					command = cmdValue
				}
				if cmdValue, ok := data["cmd"].(string); ok {
					command = cmdValue
				}

				if data["is_sudo"] != nil {
					isSudo = data["is_sudo"].(bool)
				}
			}
		}

		// We check for any of the denies in any groups. If I'm member of two groups and the command is denied by one but allowed by the other, deny wins.
		if command != "" {
			for _, CommandsRestricted := range cr {
				// * Step 1 check, if the command matches against any of the deny expressions, the command is denied
				if _, ok := CommandsRestricted["deny"].([]interface{}); ok { // we have a deny list
					if inList, whichOne := restrictionInList("deny", command, CommandsRestricted,
						func(pValue string, restriction string) bool {
							r, err := regexp.Compile(restriction)
							if err != nil {
								fmt.Printf("invalid deny restriction regular expression %q: %v", restriction, err) // ED TODO: need a validation function for the extended permissions regexes, on save
							}
							return r.Match([]byte(pValue))
						}); inList {
						return errors.New(fmt.Sprintf("Command '%v' forbidden. Allowed values must not match DENY '%v' regular expressions: %v", command, whichOne, CommandsRestricted["deny"]))
					}
				}

				// * Step 2: The command must match against any of the allow expressions. Otherwise, the command is denied.
				if _, ok := CommandsRestricted["allow"].([]interface{}); ok { // we have an allow list
					if inList, _ := restrictionInList("allow", command, CommandsRestricted,
						func(pValue string, restriction string) bool {
							r, err := regexp.Compile(restriction)
							if err != nil {
								fmt.Printf("invalid allow restriction regular expression %q: %v", restriction, err) // ED TODO: need a validation function for the extended permissions regexes, on save
							}
							return r.Match([]byte(pValue))
						}); !inList {
						return errors.New(fmt.Sprintf("Command '%v' forbidden. Allowed values must match one of ALLOW regular expressions: %v", command, CommandsRestricted["allow"]))
					}
				}

				if (isSudo) && CommandsRestricted["is_sudo"].(bool) == false {
					return errors.New(fmt.Sprintf("Command '%v' forbidden. Allowed values must not use the global is_sudo switch", command))
				}
				// if is_sudo, ok := CommandsRestricted["is_sudo"].(bool); ok {
				// 	fmt.Printf("\nis_sudo: %v and should be %v", isSudo, is_sudo)
				// }

			}

		}
	}
	return nil
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
