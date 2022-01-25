package auditlog

const (
	ActionCreate       = "create"
	ActionDelete       = "delete"
	ActionUpdate       = "update"
	ActionExecuteStart = "execute.start"
	ActionExecuteDone  = "execute.done"
)

const (
	ApplicationAuthUser        = "auth.user"
	ApplicationAuthUserMe      = "auth.user.me"
	ApplicationAuthUserMeToken = "auth.user.me.token" //nolint:gosec
	ApplicationAuthUserTotP    = "auth.user.totp"
	ApplicationClient          = "client"
	ApplicationClientACL       = "client.acl"
	ApplicationClientAuth      = "client.auth"
	ApplicationClientGroup     = "client.group"
	ApplicationClientTunnel    = "client.tunnel"
	ApplicationClientCommand   = "client.command"
	ApplicationClientScript    = "client.script"
	ApplicationLibraryCommand  = "library.command"
	ApplicationLibraryScript   = "library.script"
	ApplicationVault           = "vault"
	ApplicationSchedule        = "schedule"
)
