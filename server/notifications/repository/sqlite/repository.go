package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/realvnc-labs/rport/server/notifications"
	"github.com/realvnc-labs/rport/share/refs"
)

type Repository interface {
	//	Save(ctx context.Context, details notifications.NotificationDetails) error
	List(ctx context.Context) ([]notifications.NotificationSummary, error)
	Details(ctx context.Context, nid string) (notifications.NotificationDetails, bool, error)
	Create(ctx context.Context, details notifications.NotificationDetails) error
	LogRunning(ctx context.Context, nid string) error
	LogDone(ctx context.Context, nid string) error
	LogError(ctx context.Context, nid string, error string) error
	NotificationStream(target notifications.Target) chan notifications.NotificationDetails
	Close() error
}

const MaxNotificationsQueue = 1000

const RecipientsSeparator = "@|@"

type repository struct {
	db *sqlx.DB
	//converter *query.SQLConverter
	sinks map[notifications.Target]chan notifications.NotificationDetails
}

func (r repository) LogError(ctx context.Context, nid string, error string) error {
	return r.setState(ctx, nid, notifications.ProcessingStateError, error)
}

func (r repository) LogDone(ctx context.Context, nid string) error {
	return r.setState(ctx, nid, notifications.ProcessingStateDone, "")
}

func (r repository) LogRunning(ctx context.Context, nid string) error {
	return r.setState(ctx, nid, notifications.ProcessingStateRunning, "")
}

func (r repository) setState(ctx context.Context, nid string, state notifications.ProcessingState, out string) error {
	n := SQLNotification{
		NotificationID: nid,
		State:          string(state),
		Out:            out,
	}

	_, err := r.db.NamedExecContext(
		ctx,
		"INSERT INTO `notifications_log`"+
			" (`notification_id`, `state`, `out`)"+
			" VALUES "+
			"(:notification_id, :state, :out)",
		n,
	)

	return err
}

type SQLNotification struct {
	NotificationID string     `db:"notification_id"`
	Timestamp      *time.Time `db:"timestamp"`
	ContentType    string     `db:"contentType"`
	Origin         string     `db:"origin"`
	FullOrigin     string     `db:"fullOrigin"`
	ReferenceID    string     `db:"reference_id"`
	Transport      string     `db:"transport"`
	Recipients     string     `db:"recipients"`
	State          string     `db:"state"`
	Subject        string     `db:"subject"`
	Body           string     `db:"body"`
	Out            string     `db:"out"`
}

func (r repository) Create(ctx context.Context, details notifications.NotificationDetails) error {

	if !details.Target.Valid() {
		return fmt.Errorf("invalid target: %v", details.Target)
	}

	r.sinks[details.Target] <- details

	n := SQLNotification{
		NotificationID: details.ID.ID(),
		Timestamp:      nil,
		Origin:         string(details.Origin.Parent().Type()),
		FullOrigin:     details.Origin.String(),
		ReferenceID:    details.Origin.Parent().ID(),
		Transport:      details.Data.Target,
		Recipients:     strings.Join(details.Data.Recipients, RecipientsSeparator),
		State:          string(details.State),
		Subject:        details.Data.Subject,
		Body:           details.Data.Content,
		Out:            details.Out,
		ContentType:    string(details.Data.ContentType),
	}

	_, err := r.db.NamedExecContext(
		ctx,
		"INSERT INTO `notifications_log`"+
			" (`notification_id`, `contentType`, `origin`, `fullOrigin`, `reference_id`, `transport`, `recipients`, `state`, `subject`, `body`, `out`)"+
			" VALUES "+
			"(:notification_id, :contentType, :origin, :fullOrigin, :reference_id, :transport, :recipients, :state, :subject, :body, :out)",
		n,
	)

	return err
}

func (r repository) Details(ctx context.Context, nid string) (notifications.NotificationDetails, bool, error) {
	q := "SELECT * FROM `notifications_log` WHERE `notification_id` = ? order by oid asc"

	empty := notifications.NotificationDetails{}
	entities := []SQLNotification{}
	err := r.db.SelectContext(ctx, &entities, q, nid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return empty, false, nil
		}

		return empty, false, err
	}
	if len(entities) == 0 {
		return empty, false, nil
	}
	entity := entities[0]

	origin, err := refs.ParseOrigin(entity.FullOrigin)
	if err != nil {
		return notifications.NotificationDetails{}, false, err
	}

	var recipients []string
	if len(entity.Recipients) > 0 {
		recipients = strings.Split(entity.Recipients, RecipientsSeparator)
	}

	last := entities[len(entities)-1]
	details := notifications.NotificationDetails{
		Origin: origin,
		Data: notifications.NotificationData{
			Target:      entity.Transport,
			Recipients:  recipients,
			Subject:     entity.Subject,
			Content:     entity.Body,
			ContentType: notifications.ContentType(entity.ContentType),
		},
		State:  notifications.ProcessingState(last.State),
		ID:     refs.NewIdentifiable(notifications.NotificationType, entity.NotificationID),
		Out:    last.Out,
		Target: notifications.FigureOutTarget(entity.Transport),
	}
	tmp := details

	return tmp, true, nil
}

func (r repository) List(ctx context.Context) ([]notifications.NotificationSummary, error) {
	var res []notifications.NotificationSummary
	err := r.db.SelectContext(
		ctx,
		&res,
		"SELECT notification_id, state FROM notifications_log ORDER by notification_id",
	)
	return res, err
}

//nolint:revive
func NewRepository(connection *sqlx.DB) repository {
	sinks := map[notifications.Target]chan notifications.NotificationDetails{}
	for _, target := range notifications.AllTargets {
		sinks[target] = make(chan notifications.NotificationDetails, MaxNotificationsQueue)
	}
	return repository{
		db:    connection,
		sinks: sinks,
	}
}

func (r repository) NotificationStream(target notifications.Target) chan notifications.NotificationDetails {
	return r.sinks[target]
}

func (r repository) Close() error {
	for _, ch := range r.sinks {
		close(ch)
	}
	return nil
}
