package scriptRunner

import (
	"context"
	"encoding/json"
	"time"

	"github.com/realvnc-labs/rport/server/notifications"
)

const ScriptTimeout = time.Second * 20

type consumer struct {
}

//nolint:revive
func NewConsumer() *consumer {
	return &consumer{}
}

type ScriptIO struct {
	Data       map[string]any `json:"data"`
	Recipients []string       `json:"recipients"`
}

func (c consumer) Process(details notifications.NotificationDetails) error {
	ctx, cancelFunc := context.WithTimeout(context.Background(), ScriptTimeout)

	var content interface{} = map[string]interface{}{}
	var err error

	switch details.Data.ContentType {
	case notifications.ContentTypeTextJSON:
		err = json.Unmarshal([]byte(details.Data.Content), &content)
		if err != nil {
			cancelFunc()
			return err
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
		cancelFunc()
		return err
	}

	err = RunCancelableScript(ctx, details.Data.Target, string(data))

	cancelFunc()

	return err
}

func (c consumer) Target() notifications.Target {
	return notifications.TargetScript
}
