package rundata

import (
	"github.com/openrport/openrport/plus/capabilities/alerting/entities/clientupdates"
	"github.com/openrport/openrport/plus/capabilities/alerting/entities/measures"
	"github.com/openrport/openrport/plus/capabilities/alerting/entities/rules"
	"github.com/openrport/openrport/plus/capabilities/alerting/entities/templates"
	"github.com/openrport/openrport/plus/capabilities/alerting/entities/validations"
	"github.com/openrport/openrport/server/notifications"
	"github.com/openrport/openrport/share/refs"
)

type RunData struct {
	CL            []clientupdates.Client `json:"client_data"`
	M             []measures.Measure     `json:"measurements"`
	RS            rules.RuleSet          `json:"ruleset"`
	NT            []templates.Template   `json:"templates"`
	WaitMilliSecs int                    `json:"delay_ms"`
}

type RecordingStatus int

type SampleData struct {
	CL []clientupdates.Client `json:"client_data"`
	M  []measures.Measure     `json:"measurements"`
}

type NotificationResult struct {
	RefID        refs.Identifiable              `json:"ref_id"`
	Notification notifications.NotificationData `json:"notification"`
}

type NotificationResults []NotificationResult

type TestResults struct {
	Problems      []*rules.Problem      `json:"problems"`
	Notifications NotificationResults   `json:"notifications"`
	LogOutput     string                `json:"log_output"`
	Errs          validations.ErrorList `json:"validation_errors,omitempty"`
	Err           error                 `json:"error,omitempty"`
}
