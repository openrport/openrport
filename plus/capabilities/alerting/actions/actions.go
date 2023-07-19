package actions

type ActionType string

const (
	LogType     ActionType = "log"
	NotifyType  ActionType = "notify"
	IgnoreType  ActionType = "ignore"
	UnknownType ActionType = "unknown"
)
