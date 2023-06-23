package sqlite

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/realvnc-labs/rport/server/notifications"
	"github.com/realvnc-labs/rport/share/query"
	"github.com/realvnc-labs/rport/share/refs"
)

type Repository interface {
	//	Save(ctx context.Context, details notifications.NotificationDetails) error
	List(ctx context.Context) ([]notifications.NotificationSummary, error)
	Details(ctx context.Context, nid string) (notifications.NotificationDetails, bool, error)
	Create(background context.Context, details notifications.NotificationDetails) error
}

const RecipientsSeparator = "@|@"

type repository struct {
	db        *sqlx.DB
	converter *query.SQLConverter
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
	q := "SELECT * FROM `notifications_log` WHERE `notification_id` = ?"

	empty := notifications.NotificationDetails{}
	entity := SQLNotification{}
	err := r.db.GetContext(ctx, &entity, q, nid)
	if err != nil {
		if err == sql.ErrNoRows {
			return empty, false, nil
		}

		return empty, false, err
	}

	origin, err := refs.ParseOrigin(entity.FullOrigin)
	if err != nil {
		return notifications.NotificationDetails{}, false, err
	}

	var recipients []string
	if len(entity.Recipients) > 0 {
		recipients = strings.Split(entity.Recipients, RecipientsSeparator)
	}

	details := notifications.NotificationDetails{
		Origin: origin,
		Data: notifications.NotificationData{
			Target:      entity.Transport,
			Recipients:  recipients,
			Subject:     entity.Subject,
			Content:     entity.Body,
			ContentType: notifications.ContentType(entity.ContentType),
		},
		State:  notifications.ProcessingState(entity.State),
		ID:     refs.NewIdentifiable(notifications.NotificationType, entity.NotificationID),
		Out:    entity.Out,
		Target: notifications.FigureOutTarget(entity.Transport),
	}
	tmp := details

	return tmp, true, nil
}

func (r repository) List(ctx context.Context) ([]notifications.NotificationSummary, error) {
	return nil, nil
}

func NewRepository(connection *sqlx.DB) repository {
	return repository{
		db: connection,
	}
}
