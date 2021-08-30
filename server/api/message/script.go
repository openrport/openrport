package message

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"

	email2 "github.com/cloudradar-monitoring/rport/share/email"
)

const stderrLimit = 1024

type ValidationType string

const (
	ValidationNone  = ValidationType("none")
	ValidationEmail = ValidationType("email")
	ValidationRegex = ValidationType("regex")
)

type ScriptService struct {
	Script     string
	Validation ValidationType
	Regex      *regexp.Regexp
}

func NewScriptService(script string, validation ValidationType, regex *regexp.Regexp) *ScriptService {
	return &ScriptService{
		Script:     script,
		Validation: validation,
		Regex:      regex,
	}
}

func (s *ScriptService) Send(ctx context.Context, data Data) error {
	stderr := &bytes.Buffer{}

	cmd := exec.CommandContext(ctx, s.Script)
	cmd.Env = append(os.Environ(), s.DataToEnv(data)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("%s", stderr.Next(stderrLimit))
		}
		return err
	}

	return nil
}

func (s *ScriptService) DataToEnv(data Data) []string {
	return []string{
		fmt.Sprintf("RPORT_2FA_TOKEN=%v", data.Token),
		fmt.Sprintf("RPORT_2FA_SENDTO=%v", data.SendTo),
		fmt.Sprintf("RPORT_2FA_TOKEN_TTL=%v", data.TTL.Seconds()),
	}
}

func (s *ScriptService) DeliveryMethod() string {
	return "script"
}

func (s *ScriptService) ValidateReceiver(ctx context.Context, receiver string) error {
	switch s.Validation {
	case ValidationEmail:
		return email2.Validate(receiver)
	case ValidationRegex:
		if !s.Regex.MatchString(receiver) {
			return fmt.Errorf("does not match %q", s.Regex)
		}
		return nil
	default:
		return nil
	}
}
