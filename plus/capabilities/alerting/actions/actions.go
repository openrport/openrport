package actions

type AT string

const (
	LogActionType    AT = "log_action"
	NotifyActionType AT = "notify_action"
	IgnoreActionType AT = "ignore_action"
)
