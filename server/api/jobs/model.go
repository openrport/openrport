package jobs

import (
	"fmt"
	"strings"

	"github.com/cloudradar-monitoring/rport/server/clients"
	"github.com/cloudradar-monitoring/rport/share/models"
)

type MultiJobRequest struct {
	ClientIDs           []string              `json:"client_ids"`
	GroupIDs            []string              `json:"group_ids"`
	ClientTags          *models.JobClientTags `json:"tags"`
	Command             string                `json:"command"`
	Script              string                `json:"script"`
	Cwd                 string                `json:"cwd"`
	IsSudo              bool                  `json:"is_sudo"`
	Interpreter         string                `json:"interpreter"`
	TimeoutSec          int                   `json:"timeout_sec"`
	ExecuteConcurrently bool                  `json:"execute_concurrently"`
	AbortOnError        *bool                 `json:"abort_on_error"` // pointer is used because it's default value is true. Otherwise it would be more difficult to check whether this field is missing or not

	Username       string            `json:"-"`
	IsScript       bool              `json:"-"`
	OrderedClients []*clients.Client `json:"-"`
	ScheduleID     *string           `json:"-"`
}

func (req *MultiJobRequest) GetClientIDs() (ids []string) {
	return req.ClientIDs
}

func (req *MultiJobRequest) GetGroupIDs() (ids []string) {
	return req.GroupIDs
}

func (req *MultiJobRequest) GetClientTags() (clientTags *models.JobClientTags) {
	return req.ClientTags
}

// TODO: add some unit tests. not high priority but good to get done.
func MakeClientTagsAsString(jobTags *models.JobClientTags) (clientTags string) {
	if jobTags == nil {
		return "[]"
	}
	numTags := len(jobTags.Tags)
	if numTags == 0 {
		return "[]"
	}
	tagsList := strings.Join(jobTags.Tags, ",")
	if numTags == 1 {
		clientTags = fmt.Sprintf("[%s]", tagsList)
	} else {
		operator := jobTags.Operator
		if operator == "" {
			operator = "OR"
		}
		clientTags = fmt.Sprintf("[%s: %s]", operator, tagsList)
	}
	return clientTags
}
