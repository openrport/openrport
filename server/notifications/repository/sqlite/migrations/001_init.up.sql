CREATE TABLE notifications_log (
    notification_id CHAR(26) NOT NULL CHECK (notification_id != ''), -- notification_id, number of the log entry, ulid
    timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,    -- timestamp, date and time of message dispatching
    contentType VARCHAR(50)  NOT NULL DEFAULT "",
    reference_id VARCHAR(26)  NOT NULL DEFAULT "",   -- refrence_id, problem id handed over by the altering service.
    transport TEXT NOT NULL DEFAULT "",                                           -- transport, either "smtp" or name of the script
    recipients TEXT NOT NULL DEFAULT "",                                         -- recipients, serialized array

    state VARCHAR(20) NOT NULL CHECK (state != ''), -- sqlite doesn't like enums

    subject TEXT NOT NULL DEFAULT "",
    body TEXT NOT NULL DEFAULT "",

    out TEXT NOT NULL DEFAULT "", -- whatever error is returned only when error is returned
    err TEXT NOT NULL DEFAULT "" -- whatever error is returned only when error is returned
);
--

CREATE INDEX idx_notifications_id
ON notifications_log (notification_id);

CREATE INDEX idx_notifications_timestamp
    ON notifications_log (timestamp);

-- status_code, smtp status code returned by the SMTP server or the exit code of the script
-- smtp_errors, error message returned from the SMTP server
-- script_response, first 2k of stdout and stderr of the script