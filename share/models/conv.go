package models

func StrToBool(input string) bool {
	switch input {
	case "":
		return false
	case "0":
		return false
	case "false":
		return false
	default:
		return true
	}
}
