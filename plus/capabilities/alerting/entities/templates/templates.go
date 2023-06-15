package templates

import "errors"

type TemplateID string

type Transport string

const (
	SMTP   Transport = "smtp"
	Script Transport = "script"
)

var (
	ErrTemplateNotFound = errors.New("template not found")
)

type Template struct {
	ID         TemplateID `mapstructure:"id" json:"id"`
	Transport  Transport  `mapstructure:"transport" json:"transport"`
	Subject    string     `mapstructure:"subject" json:"subject"`
	Body       string     `mapstructure:"body" json:"body"`
	HTML       bool       `mapstructure:"html" json:"html"`
	Recipients []string   `mapstructure:"recipients" json:"recipients"`
}

type TemplateList []*Template
