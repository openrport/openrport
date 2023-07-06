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
	"github.com/realvnc-labs/rport/share/logger"
	"github.com/realvnc-labs/rport/share/query"
	"github.com/realvnc-labs/rport/share/refs"
)

type Repository interface {
	List(ctx context.Context, options *query.ListOptions) ([]notifications.NotificationSummary, error)
	Count(ctx context.Context, options *query.ListOptions) (int, error)
	Details(ctx context.Context, nid string) (notifications.NotificationDetails, bool, error)
	Create(ctx context.Context, details notifications.NotificationDetails) error
	SetDone(ctx context.Context, details notifications.NotificationDetails) error
	SetError(ctx context.Context, details notifications.NotificationDetails, out string) error
	NotificationStream(target notifications.Target) chan notifications.NotificationDetails
	Close() error
}

const MaxNotificationsQueue = 1000

const RecipientsSeparator = "@|@"

type repository struct {
	db        *sqlx.DB
	converter *query.SQLConverter
	sinks     map[notifications.Target]chan notifications.NotificationDetails

	L *logger.Logger
}

func (r repository) Count(ctx context.Context, options *query.ListOptions) (int, error) {
	var res int

	countOptions := *options
	countOptions.Pagination = nil
	q := "SELECT COUNT(*) FROM notifications_log ORDER by notification_id"
	params := []interface{}{}
	q, params = r.converter.AppendOptionsToQuery(&countOptions, q, params)

	err := r.db.SelectContext(
		ctx,
		&res,
		q,
		params...,
	)
	return res, err
}

func (r repository) SetError(ctx context.Context, details notifications.NotificationDetails, out string) error {
	details.Out = out
	details.State = notifications.ProcessingStateError
	return r.save(ctx, details)
}

func (r repository) SetDone(ctx context.Context, details notifications.NotificationDetails) error {
	details.State = notifications.ProcessingStateDone
	return r.save(ctx, details)
}

type SQLNotification struct {
	NotificationID string     `db:"notification_id"`
	Timestamp      *time.Time `db:"timestamp"`
	ContentType    string     `db:"contentType"`
	ReferenceID    string     `db:"reference_id"`
	Transport      string     `db:"transport"`
	Recipients     string     `db:"recipients"`
	State          string     `db:"state"`
	Subject        string     `db:"subject"`
	Body           string     `db:"body"`
	Out            string     `db:"out"`
}

func (r repository) Create(_ context.Context, details notifications.NotificationDetails) error {

	if !details.Target.Valid() {
		return fmt.Errorf("invalid target: %v", details.Target)
	}

	ch := r.sinks[details.Target]
	if len(ch) > MaxNotificationsQueue*0.95 {
		return fmt.Errorf("notification rejected due to too many queued notifications: %v", details.Target)
	}

	ch <- details

	return nil
}

func (r repository) save(ctx context.Context, details notifications.NotificationDetails) error {

	n := SQLNotification{
		NotificationID: details.ID.ID(),
		Timestamp:      nil,
		ReferenceID:    details.RefID.String(),
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
		"INSERT INTO `notifications_log` "+
			" (`notification_id`, `contentType`, `reference_id`, `transport`, `recipients`, `state`, `subject`, `body`, `out`)"+
			" VALUES "+
			"(:notification_id, :contentType, :reference_id, :transport, :recipients, :state, :subject, :body, :out)",
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

	refID, err := refs.ParseIdentifiable(entity.ReferenceID)
	if err != nil {
		return notifications.NotificationDetails{}, false, err
	}

	var recipients []string
	if len(entity.Recipients) > 0 {
		recipients = strings.Split(entity.Recipients, RecipientsSeparator)
	}

	last := entities[len(entities)-1]
	details := notifications.NotificationDetails{
		RefID: refID,
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

func (r repository) List(ctx context.Context, options *query.ListOptions) ([]notifications.NotificationSummary, error) {
	var res []notifications.NotificationSummary

	q := `
SELECT notification_id, state, transport, timestamp, out
FROM notifications_log ORDER by timestamp desc`
	params := []interface{}{}
	q, params = r.converter.AppendOptionsToQuery(options, q, params)

	err := r.db.SelectContext(
		ctx,
		&res,
		q,
		params...,
	)
	return res, err
}

//nolint:revive
func NewRepository(connection *sqlx.DB, l *logger.Logger) repository {
	sinks := map[notifications.Target]chan notifications.NotificationDetails{}
	for _, target := range notifications.AllTargets {
		sinks[target] = make(chan notifications.NotificationDetails, MaxNotificationsQueue)
	}
	return repository{
		db:        connection,
		sinks:     sinks,
		converter: query.NewSQLConverter(connection.DriverName()),
		L:         l,
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
