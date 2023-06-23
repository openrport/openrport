CREATE TABLE notifications_log (
    notification_id CHAR(26) NOT NULL CHECK (notification_id != ''), -- notification_id, number of the log entry, ulid
    timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,    -- timestamp, date and time of message dispatching
    origin VARCHAR(20) NOT NULL CHECK (origin != ''),         -- origin, name of the subsystem that has created the notification
    fullOrigin VARCHAR(255) NOT NULL CHECK (fullOrigin != ''),
    contentType VARCHAR(50) NOT NULL CHECK (contentType != ''),
    reference_id VARCHAR(26) NOT NULL CHECK (reference_id != ''),   -- refrence_id, problem id handed over by the altering service.
    transport TEXT,                                           -- transport, either "smtp" or name of the script
    recipients TEXT,                                         -- recipients, serialized array

    state VARCHAR(20) NOT NULL CHECK (state != ''), -- sqlite doesn't like enums

    subject TEXT,
    body TEXT,

    out TEXT, -- whatever error is returned only when error is returned

    PRIMARY KEY (`notification_id`, timestamp)

) WITHOUT ROWID;
--

CREATE UNIQUE INDEX idx_notifications_id
ON notifications_log (notification_id);

CREATE UNIQUE INDEX idx_notifications_timestamp
    ON notifications_log (timestamp);

-- status_code, smtp status code returned by the SMTP server or the exit code of the script
-- smtp_errors, error message returned from the SMTP server
-- script_response, first 2k of stdout and stderr of the script