package rundata

import (
	"github.com/realvnc-labs/rport/plus/capabilities/alerting/entities/clientupdates"
	"github.com/realvnc-labs/rport/plus/capabilities/alerting/entities/measures"
	"github.com/realvnc-labs/rport/plus/capabilities/alerting/entities/rules"
	"github.com/realvnc-labs/rport/plus/capabilities/alerting/entities/templates"
	"github.com/realvnc-labs/rport/plus/capabilities/alerting/entities/validations"
	"github.com/realvnc-labs/rport/server/notifications"
	"github.com/realvnc-labs/rport/share/refs"
)

type ClientUpdateWithWait struct {
	ClientUpdate  clientupdates.Client `json:"CL"`
	WaitMilliSecs int                  `json:"W"`
}

type MeasureWithWait struct {
	MeasureUpdate measures.Measure `json:"M"`
	WaitMilliSecs int              `json:"W"`
}

type RunData struct {
	CL []ClientUpdateWithWait `json:"client_updates"`
	M  []MeasureWithWait      `json:"measures"`
	RS rules.RuleSet          `json:"ruleset"`
	NT []templates.Template   `json:"templates"`
}

type NotificationResult struct {
	RefID        refs.Identifiable              `json:"ref_id"`
	Notification notifications.NotificationData `json:"notifications"`
}

type NotificationResults []NotificationResult

type TestResults struct {
	Problems      []*rules.Problem      `json:"problems"`
	Notifications NotificationResults   `json:"notifications"`
	LogOutput     string                `json:"log_output"`
	Errs          validations.ErrorList `json:"validation_errors"`
	Err           error                 `json:"err"`
}
