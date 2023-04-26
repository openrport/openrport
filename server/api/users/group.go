package users

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

const (
	Administrators = "Administrators"
)

var AdministratorsGroup = Group{
	Name:        Administrators,
	Permissions: NewPermissions(AllPermissions...),
}

/*
	"tunnels_restricted": {
	    "local": ["20000","20001"], 		// The user can only create tunnels that would use port 2000 or 20001 on the rport server.
	    "remote": ["22","3389"],			// The user can only create tunnels to the remote ports 22 or 3389.
	    "scheme": ["ssh","rdp"],			// Scheme must be SSH or RDP
	    "acl": ["201.203.40.9"],			// The user can only create tunnels to the remote IP
	    "min-idle-timeout-minutes": 5,		// The user can only create tunnels with an idle timeout of 5 minutes or more.
	    "max-auto-close": "60m", 			// Auto-close must be used, with a maximum of 60m, that means the user will not be able to use the tunnel for more than 60 minutes.
	    "protocol": ["tcp","udp","tcp-udp"],// Any protocols are allowed.
	    "skip-idle-timeout": 0, 			// The user is not allowed to enable skip-idle-timeout
	    "http_proxy": true, 				// The user is allowed to use the HTTP proxy
	    "host_header": ":*", 				// The user can only add a host header matching the regular expression (any host in this case)
	    "auth_allowed": true 				// The user is allowed to enable http basic auth for a tunnel
	}
*/
type TunnelsRestricted struct {
	Local           []string `json:"local,omitempty"`
	Remote          []string `json:"remote,omitempty"`
	Scheme          []string `json:"scheme,omitempty"`
	ACL             []string `json:"acl,omitempty"`
	MinIdleTimeout  int      `json:"min-idle-timeout-minutes,omitempty"`
	MaxAutoClose    string   `json:"max-auto-close,omitempty"`
	Protocol        []string `json:"protocol,omitempty"`
	SkipIdleTimeout int      `json:"skip-idle-timeout,omitempty"`
	HTTPProxy       bool     `json:"http_proxy,omitempty"`
	HostHeader      string   `json:"host_header,omitempty"`
	AuthAllowed     bool     `json:"auth_allowed,omitempty"`
}

func (t *TunnelsRestricted) Scan(value interface{}) error {
	if t == nil {
		return errors.New("'TunnelsRestricted' cannot be nil")
	}
	valueStr, ok := value.(string)
	if !ok {
		return fmt.Errorf("expected to have string, got %T", value)
	}
	err := json.Unmarshal([]byte(valueStr), t)
	if err != nil {
		return fmt.Errorf("failed to decode 'TunnelsRestricted' field: %v", err)
	}
	return nil
}

func (t *TunnelsRestricted) Value() (driver.Value, error) {
	b, err := json.Marshal(t)
	if err != nil {
		return nil, fmt.Errorf("failed to encode 'TunnelsRestricted' field: %v", err)
	}
	return string(b), nil
}

/*
	"commands_restricted": {
	    "allow": ["^sudo reboot$","^systemctl .* restart$"],		// I can reboot the machine. I can restart any service.
	    "deny": ["apache2","ssh"],									// I can restart any service except apache2 and ssh
	    "is_sudo": false											// I cannot use the global is_sudo switch. I can still prefix commands with sudo, if the keyword "sudo" is allowed.
	}
*/

/*
EDTODO: The list of deny and allow keywords are regular expressions.
Step 1: If the command matches against any of the deny expressions, the command is denied.
Step 2: The command must match against any of the allow expressions. Otherwise, the command is denied.
*/
type CommandsRestricted struct {
	Allow  []string `json:"allow,omitempty"` // EDTODO: This is a regex, Using an empty list or omitting an object will remove any restrictions. For example, if allowed is not present, or if "allowed": [] then any command can be used.
	Deny   []string `json:"deny,omitempty"`  // EDTODO: If deny is missing or empty, the command is not validated against the deny patterns.
	IsSudo bool     `json:"is_sudo,omitempty"`
}

func (c *CommandsRestricted) Scan(value interface{}) error {
	if c == nil {
		return errors.New("'CommandsRestricted' cannot be nil")
	}
	valueStr, ok := value.(string)
	if !ok {
		return fmt.Errorf("expected to have string, got %T", value)
	}
	err := json.Unmarshal([]byte(valueStr), c)
	if err != nil {
		return fmt.Errorf("failed to decode 'CommandsRestricted' field: %v", err)
	}
	return nil
}

func (c *CommandsRestricted) Value() (driver.Value, error) {
	b, err := json.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("failed to encode 'CommandsRestricted' field: %v", err)
	}
	return string(b), nil
}

// EDTODO: If the user groups have, for example, tunnel permissions and tunnels_restricted permissions, wider permissions wins. That means, to effectively enable restricted tunnels or commands, the general tunnel or commands permission must be authorized.
type Group struct {
	Name               string              `json:"name" db:"name"`
	Permissions        Permissions         `json:"permissions" db:"permissions"`
	TunnelsRestricted  *TunnelsRestricted  `json:"tunnels_restricted" db:"tunnels_restricted"`
	CommandsRestricted *CommandsRestricted `json:"commands_restricted" db:"commands_restricted"`
}

func NewGroup(name string, tr *TunnelsRestricted, cr *CommandsRestricted, perms ...string) Group {
	fmt.Println("NewGroup", name, perms)
	if name == Administrators {
		return AdministratorsGroup
	}
	return Group{
		Name:               name,
		Permissions:        NewPermissions(perms...),
		TunnelsRestricted:  tr,
		CommandsRestricted: cr,
	}
}
