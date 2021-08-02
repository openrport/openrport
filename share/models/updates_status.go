package models

import "time"

type UpdatesStatus struct {
	Refreshed                time.Time       `json:"refreshed"`
	UpdatesAvailable         int             `json:"updates_available"`
	SecurityUpdatesAvailable int             `json:"security_updates_available"`
	UpdateSummaries          []UpdateSummary `json:"update_summaries"`
	RebootPending            bool            `json:"reboot_pending"`
	Error                    string          `json:"error,omitempty"`
	Hint                     string          `json:"hint,omitempty"`
}

type UpdateSummary struct {
	Title            string `json:"title"`
	Description      string `json:"description"`
	RebootRequired   bool   `json:"reboot_required"`
	IsSecurityUpdate bool   `json:"is_security_update"`
}
