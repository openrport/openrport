package actions

type AT string

const (
	LogActionType     AT = "log"
	NotifyActionType  AT = "notify"
	IgnoreActionType  AT = "ignore"
	UnknownActionType AT = "unknown"
)
