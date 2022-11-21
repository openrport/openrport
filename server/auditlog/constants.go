package auditlog

const (
	ActionCreate       = "create"
	ActionDelete       = "delete"
	ActionUpdate       = "update"
	ActionExecuteStart = "execute.start"
	ActionExecuteDone  = "execute.done"
	ActionSuccess      = "success"
	ActionFailed       = "failed"
)

const (
	ApplicationAuthUser        = "auth.user"
	ApplicationAuthUserMe      = "auth.user.me"
	ApplicationAuthUserMeToken = "auth.user.me.token" //nolint:gosec
	ApplicationAuthUserTotP    = "auth.user.totp"
	ApplicationAuthUserGroup   = "auth.user.group"
	ApplicationAuthAPISession  = "auth.api.session"
	ApplicationAuthAPISessions = "auth.api.sessions"
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
	ApplicationUploads         = "uploads"
)
