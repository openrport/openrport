package vault

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
