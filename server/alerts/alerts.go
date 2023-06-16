package alerts

import (
	"net/http"
	"sort"
	"strings"

	"github.com/realvnc-labs/rport/plus/capabilities/alerting/entities/rules"
	"github.com/realvnc-labs/rport/server/api/errors"
	"github.com/realvnc-labs/rport/share/query"
)

var ProblemsOptionsListDefaultFields = map[string][]string{
	"fields[problems]": {
		"problem_id",
		"rule_id",
		"client_id",
		"state",
		"created_at",
		"resolved_at",
	},
}

var ProblemsOptionsSupportedSorts = map[string]bool{
	"problem_id":  false,
	"rule_id":     true,
	"client_id":   true,
	"state":       true,
	"created_at":  true,
	"resolved_at": true,
}

var ProblemsOptionsSupportedFilters = map[string]bool{
	"problem_id":  true,
	"rule_id":     true,
	"client_id":   true,
	"state":       true,
	"created_at":  true,
	"resolved_at": true,
}

var ProblemsOptionsSupportedFields = map[string]map[string]bool{
	"problems": {
		"problem_id":  true,
		"rule_id":     true,
		"client_id":   true,
		"state":       true,
		"created_at":  true,
		"resolved_at": true,
	},
}

func ProblemsSortByRuleID(problems []*rules.Problem, desc bool) {
	sort.Slice(problems, func(i, j int) bool {
		less := strings.ToLower(string(problems[i].RuleID)) < strings.ToLower(string(problems[j].RuleID))
		if desc {
			return !less
		}
		return less
	})
}

func ProblemsSortByClientID(problems []*rules.Problem, desc bool) {
	sort.Slice(problems, func(i, j int) bool {
		less := strings.ToLower(problems[i].ClientID) < strings.ToLower(problems[j].ClientID)
		if desc {
			return !less
		}
		return less
	})
}

func ProblemsSortByState(problems []*rules.Problem, desc bool) {
	sort.Slice(problems, func(i, j int) bool {
		less := strings.ToLower(string(problems[i].State)) < strings.ToLower(string(problems[j].State))
		if desc {
			return !less
		}
		return less
	})
}

func ProblemsSortByCreatedAt(problems []*rules.Problem, desc bool) {
	sort.Slice(problems, func(i, j int) bool {
		less := problems[i].CreatedAt.Before(problems[j].CreatedAt)
		if desc {
			return !less
		}
		return less
	})
}

func ProblemsSortByResolvedAt(problems []*rules.Problem, desc bool) {
	sort.Slice(problems, func(i, j int) bool {
		less := problems[i].ResolvedAt.Before(problems[j].ResolvedAt)
		if desc {
			return !less
		}
		return less
	})
}

func GetProblemsSortFunc(sorts []query.SortOption) (sortFunc func(a []*rules.Problem, desc bool), desc bool, err error) {
	if len(sorts) < 1 {
		return ProblemsSortByCreatedAt, true, nil
	}
	if len(sorts) > 1 {
		return nil, false, errors.APIError{
			Message:    "Only one sort field is supported for problems.",
			HTTPStatus: http.StatusBadRequest,
		}
	}

	switch sorts[0].Column {
	case "rule_id":
		sortFunc = ProblemsSortByRuleID
	case "client_id":
		sortFunc = ProblemsSortByClientID
	case "state":
		sortFunc = ProblemsSortByState
	case "created_at":
		sortFunc = ProblemsSortByCreatedAt
	case "resolved_at":
		sortFunc = ProblemsSortByResolvedAt
	}

	return sortFunc, !sorts[0].IsASC, nil
}
