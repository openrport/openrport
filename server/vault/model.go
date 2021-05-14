package vault

import "time"

const (
	DbStatusInit    = "setup-completed"
	DbStatusNotInit = "uninitialized"
	StatusLocked    = "locked"
	StatusUnlocked  = "unlocked"
)

type DbStatus struct {
	ID            int    `db:"id"`
	StatusName    string `db:"db_status"`
	EncCheckValue string `db:"enc_check"`
	DecCheckValue string `db:"dec_check"`
}

type StatusReport struct {
	InitStatus string `json:"init"`
	LockStatus string `json:"status"`
}

type PassRequest struct {
	Password string `json:"password"`
}

type ValueType string

const TextType ValueType = "text"
const SecreteType ValueType = "secrete"
const MarkdownType ValueType = "markdown"
const StringType ValueType = "string"

type InputValue struct {
	ClientID      string    `json:"client_id" db:"client_id"`
	RequiredGroup string    `json:"required_group" db:"required_group"`
	Key           string    `json:"key" db:"key"`
	Value         string    `json:"value" db:"value"`
	Type          ValueType `json:"type" db:"type"`
}

type ValueKey struct {
	ID        int       `db:"id"`
	ClientID  string    `json:"client_id" db:"client_id"`
	CreatedBy string    `db:"created_by"`
	CreatedAt time.Time `db:"created_at"`
	Key       string    `json:"key" db:"key"`
}

type StoredValue struct {
	InputValue
	ID        int       `db:"id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
	CreatedBy string    `db:"created_by"`
	UpdatedBy string    `db:"updated_by"`
}

type SortOption struct {
	Column string
	IsASC  bool
}

type FilterOption struct {
	Column string
	Values []string
}

type ListOptions struct {
	Sorts   []SortOption
	Filters []FilterOption
}
