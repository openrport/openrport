package notifications

type NotificationType string

const (
	EmailType  NotificationType = "email"
	ScriptType NotificationType = "script"
)

type RecipientList []string
