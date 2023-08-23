package templates

import (
	"errors"
)

var (
	ErrTemplateValidationFailed = errors.New("template validation failed")
	ErrTemplateInUse            = errors.New("template is being used by running rules. please update your rules before deleting")

	ErrMissingTemplateIDMsg                   = "missing template id"
	ErrMissingTransportMsg                    = "missing transport"
	ErrSubjectOrBodyMustBeSpecifiedMsg        = "subject or body must be specified"
	ErrMissingRecipientsMsg                   = "missing recipients"
	ErrScriptDataCannotBeSpecifiedWhenSMTPMsg = "data cannot be specified when using smtp"
	ErrMissingScriptDataMsg                   = "data must be specified when using notification scripts"
	ErrMissingScriptSubjectMsg                = "missing data subject"
	ErrBadlyFormedWebhookMsg                  = "badly formed webhook"
	ErrMissingWebhookURLHostMsg               = "missing host in webhook url"
	ErrScriptNotFoundMsg                      = "script %s not found in %s"
	ErrFailedToStatScriptFile                 = "failed to stat script file %s"
	ErrScriptNotExecutableMsg                 = "script %s not executable"
)

type TemplateID string

type CustomData map[string]string

type ScriptDataTemplates struct {
	Subject    string     `json:"subject"`
	Severity   string     `json:"severity"`
	Client     string     `json:"client"`
	WebhookURL string     `json:"webhook_url"`
	Custom     CustomData `json:"custom_data"`
}

type Template struct {
	ID                  TemplateID           `json:"id"`
	Transport           string               `json:"transport"`
	Subject             string               `json:"subject,omitempty"`
	Body                string               `json:"body,omitempty"`
	HTML                bool                 `json:"html"`
	ScriptDataTemplates *ScriptDataTemplates `json:"data,omitempty"`
	Recipients          []string             `json:"recipients,omitempty"`
}

type TemplateList []*Template
