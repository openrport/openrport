package scriptRunner

import (
	"context"
	"encoding/json"
	"time"

	"github.com/realvnc-labs/rport/server/notifications"
	"github.com/realvnc-labs/rport/share/logger"
)

const ScriptTimeout = time.Second * 20

type consumer struct {
	l          *logger.Logger
	workingDir string
}

//nolint:revive
func NewConsumer(l *logger.Logger, workingDir string) *consumer {
	return &consumer{
		l:          l,
		workingDir: workingDir,
	}
}

func (c consumer) Process(ctx context.Context, details notifications.NotificationDetails) (string, error) {
	ctx, cancelFunc := context.WithTimeout(ctx, ScriptTimeout)
	defer cancelFunc()

	var content interface{} = map[string]interface{}{}
	var err error

	switch details.Data.ContentType {
	case notifications.ContentTypeTextJSON:
		err = json.Unmarshal([]byte(details.Data.Content), &content)
		if err != nil {
			return "", err
		}
	default:
		content = details.Data.Content
	}

	tmp := map[string]interface{}{
		"recipients": details.Data.Recipients,
		"data":       content,
	}

	data, err := json.Marshal(&tmp)
	if err != nil {
		return "", err
	}

	c.l.Debugf("running script: %s: with data: %s", details.Data.Target, string(data))

	out, err := RunCancelableScript(ctx, c.workingDir, details.Data.Target, string(data))
	if err != nil {
		c.l.Debugf("failed running script: %s: with err: ", details.Data.Target, err)
		return out, err
	}

	return out, nil
}

func (c consumer) Target() notifications.Target {
	return notifications.TargetScript
}
