package chserver

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"testing"
	"time"

	smtpmock "github.com/mocktools/go-smtp-mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"

	"github.com/openrport/openrport/server/api/message"
	"github.com/openrport/openrport/server/api/users"
)

type usrService struct {
	UserService
	twoFASendTo string
}

func (u usrService) GetByUsername(username string) (*users.User, error) {
	return &users.User{
		Username:    username,
		TwoFASendTo: u.twoFASendTo,
	}, nil
}

const tokenTTL = 100

func TestSMTPMessage(t *testing.T) {
	// Start an SMTP server
	smtpSrv := smtpmock.New(smtpmock.ConfigurationAttr{
		LogToStdout:       false,
		LogServerActivity: false,
	})
	err := smtpSrv.Start()
	t.Cleanup(func() {
		err := smtpSrv.Stop()
		require.NoError(t, err)
	})
	require.NoError(t, err)
	msgSrv, err := message.NewSMTPService(
		fmt.Sprintf("127.0.0.1:%d", smtpSrv.PortNumber),
		"",
		"",
		"rport@example.com",
		false,
	)
	require.NoError(t, err)
	// Send the message using the service to be tested
	srv := NewTwoFAService(tokenTTL, 2*time.Second, &usrService{twoFASendTo: "j@example.com"}, msgSrv)
	ctx := context.Background()
	sendTo, err := srv.SendToken(ctx, "John", "Go", "127.0.0.1")
	assert.Contains(t, sendTo, "j@example.com")
	require.NoError(t, err)

	// Read email caught by the smtp server
	msgBody := smtpSrv.Messages()[0].MsgRequest()
	assert.Regexp(t, "The token is:[a-z A-Z 0-9]{6}", msgBody)
	assert.Contains(t, msgBody, "with user agent Go")
	assert.Contains(t, msgBody, "The token has been requested from 127.0.0.1")
	assert.Contains(t, msgBody, fmt.Sprintf("Token is valid for %d seconds", tokenTTL))
	assert.Contains(t, smtpSrv.Messages()[0].RcpttoRequest(), sendTo)
}

func TestScriptMessage(t *testing.T) {
	tmpDir := t.TempDir() + "/"
	tfaScript := struct {
		name, log string
		content   []byte
		mode      os.FileMode
	}{
		name:    tmpDir + "tfa.sh",
		log:     tmpDir + "tfa.log",
		content: []byte("#!/bin/sh\nset|grep RPORT_>" + tmpDir + "tfa.log"),
		mode:    0700,
	}

	err := os.WriteFile(tfaScript.name, tfaScript.content, tfaScript.mode)
	require.NoError(t, err, "Writing script failed.")
	regex := regexp.MustCompile(".*")
	msgSrv := message.NewScriptService(
		tfaScript.name,
		message.ValidationNone,
		regex,
	)
	// Send the message using the service to be tested
	srv := NewTwoFAService(tokenTTL, 2*time.Second, &usrService{twoFASendTo: "t@example.com"}, msgSrv)
	sendTo, err := srv.SendToken(context.Background(), "Tilda", "Go", "127.0.0.1")
	require.NoError(t, err, "message service sendToken returns error")
	assert.Contains(t, sendTo, "t@example.com")

	// Read the output of the script which contains the entire shell environment
	buf, err := os.ReadFile(tfaScript.log)
	scriptLog := string(buf)
	require.NoError(t, err)
	assert.Regexp(t, regexp.MustCompile(fmt.Sprintf("RPORT_2FA_SENDTO='*%s'*", "t@example.com")), scriptLog)
	assert.Regexp(t, regexp.MustCompile("RPORT_2FA_REMOTE_ADDRESS='*127.0.0.1'*"), scriptLog)
	assert.Regexp(t, regexp.MustCompile("RPORT_2FA_USER_AGENT='*Go'*"), scriptLog)
	assert.Regexp(t, regexp.MustCompile("RPORT_2FA_TOKEN='*[a-z A-Z 0-9]{6}'*"), scriptLog)
	assert.Regexp(t, regexp.MustCompile(fmt.Sprintf("RPORT_2FA_TOKEN_TTL='*%d'*", tokenTTL)), scriptLog)

	// Delete script and its output
	err = os.Remove(tfaScript.name)
	require.NoError(t, err)
	err = os.Remove(tfaScript.log)
	require.NoError(t, err)
}

func TestPushoverMessage(t *testing.T) {
	// Mock the Pushover API
	defer gock.Off() // Flush pending mocks after test execution
	gock.Observe(func(request *http.Request, mock gock.Mock) {
		// Verify the post request contains our data
		bytes, _ := httputil.DumpRequest(request, true)
		body, err := url.QueryUnescape(string(bytes))
		require.NoError(t, err)
		assert.Contains(t, body, "requested from 127.0.0.1")
		assert.Contains(t, body, "with Go")
		assert.Contains(t, body, fmt.Sprintf("valid for %d seconds", tokenTTL))
		assert.Regexp(t, regexp.MustCompile("Your RPort 2fa token:<b>[a-z A-Z 0-9]{6}</b>"), body)
	})

	gock.New("https://api.pushover.net/1").
		Reply(200).
		BodyString(`{"status":1,"request":"d2dc4e70-9daf-4fd6-b07a-7065915eceb9"}`).
		SetHeader("x-limit-app-limit", "1").
		SetHeader("x-limit-app-remaining", "1").
		SetHeader("x-limit-app-reset", "9999999999")

	// Send the message using the service to be tested
	msgSrv := message.NewPushoverService("Aec5oohoa7aePooTee1sheesae1lei")
	srv := NewTwoFAService(tokenTTL, 2*time.Second, &usrService{twoFASendTo: "uo2t5thob2f1cbpssn6a8zq7i2bdka"}, msgSrv)
	sendTo, err := srv.SendToken(context.Background(), "Wendy", "Go", "127.0.0.1")
	require.NoError(t, err, "message service sendToken returns error")
	assert.Contains(t, sendTo, "uo2t5thob2f1cbpssn6a8zq7i2bdka")
}
