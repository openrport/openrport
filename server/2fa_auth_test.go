package chserver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/require"

	"github.com/cloudradar-monitoring/rport/server/api/message"
	"github.com/cloudradar-monitoring/rport/server/api/users"
)

func TestValidateTotPCode(t *testing.T) {
	usr := &users.User{
		Username: "no@mail.me",
	}
	usrService := &MockUsersService{}
	msgSErvice := &message.ServiceMock{}

	tfaService := NewTwoFAService(100, time.Second, usrService, msgSErvice)
	tfaService.SetTotPLoginSession(usr.Username, time.Minute)

	inpt := &TotPInput{
		Issuer:      "iss1",
		AccountName: "acc1",
	}
	totP, err := GenerateTotPSecretKey(inpt)
	require.NoError(t, err)

	StoreTotPCodeInUser(usr, totP)

	code, err := totp.GenerateCode(totP.Secret, time.Now())
	require.NoError(t, err)

	err = tfaService.ValidateTotPCode(usr, code)
	require.NoError(t, err)

	err = tfaService.ValidateTotPCode(usr, "dfasdf")
	require.EqualError(t, err, "login request not found for provided username")
}

func TestInvalidTotPCode(t *testing.T) {
	usr := &users.User{
		Username: "no1@mail.me",
	}

	tfaService := NewTwoFAService(100, time.Second, &MockUsersService{}, &message.ServiceMock{})
	tfaService.SetTotPLoginSession(usr.Username, time.Minute)

	inpt := &TotPInput{
		Issuer:      "iss2",
		AccountName: "acc2",
	}
	totP, err := GenerateTotPSecretKey(inpt)
	require.NoError(t, err)

	StoreTotPCodeInUser(usr, totP)

	err = tfaService.ValidateTotPCode(usr, "dfasdf")
	require.EqualError(t, err, "invalid codem")
}

func TestValidateTotPCodeLoginExpired(t *testing.T) {
	usr := &users.User{
		Username: "no@mail.me",
	}
	usrService := &MockUsersService{}
	msgSErvice := &message.ServiceMock{}

	tfaService := NewTwoFAService(100, time.Second, usrService, msgSErvice)
	tfaService.SetTotPLoginSession(usr.Username, -100)

	err := tfaService.ValidateTotPCode(usr, "123")
	require.EqualError(t, err, "login request expired")
}

func TestValidateTotPCodeInvalidTotPData(t *testing.T) {
	usr := &users.User{
		Username: "no@mail.me",
		TotP:     "w43dfa",
	}
	usrService := &MockUsersService{}
	msgSErvice := &message.ServiceMock{}

	tfaService := NewTwoFAService(100, time.Second, usrService, msgSErvice)
	tfaService.SetTotPLoginSession(usr.Username, time.Minute)

	err := tfaService.ValidateTotPCode(usr, "123")
	assert.Contains(t, err.Error(), "failed to convert 'w43dfa' to TotP secret data")

	usr2 := &users.User{
		Username: "no@mail.me",
		TotP:     "",
	}
	err = tfaService.ValidateTotPCode(usr2, "123")
	require.EqualError(t, err, "time based one time secret key should be generated for this user")
}

func TestValidateTotPCodeWithUnknownUser(t *testing.T) {
	usr := &users.User{}
	usrService := &MockUsersService{}
	msgSErvice := &message.ServiceMock{}

	tfaService := NewTwoFAService(100, time.Second, usrService, msgSErvice)

	err := tfaService.ValidateTotPCode(usr, "123")
	require.EqualError(t, err, "login request not found for provided username")
}
