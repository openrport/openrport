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
)

type TemplateID string

type ScriptDataTemplates struct {
	Subject    string `json:"subject"`
	Severity   string `json:"severity"`
	Client     string `json:"client"`
	WebhookURL string `json:"webhook_url"`
}

type Template struct {
	ID                  TemplateID           `mapstructure:"id" json:"id"`
	Transport           string               `mapstructure:"transport" json:"transport"`
	Subject             string               `mapstructure:"subject" json:"subject,omitempty"`
	Body                string               `mapstructure:"body" json:"body,omitempty"`
	HTML                bool                 `mapstructure:"html" json:"html"`
	ScriptDataTemplates *ScriptDataTemplates `mapstructure:"data" json:"data,omitempty"`
	Recipients          []string             `mapstructure:"recipients" json:"recipients,omitempty"`
}

type TemplateList []*Template
