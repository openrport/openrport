package chserver

import (
	"bytes"
	"encoding/base64"
	"image/png"
	"testing"
	"time"

	"github.com/cloudradar-monitoring/rport/server/api/users"

	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCodeGenerationAndValidation(t *testing.T) {
	inpt := &TotPInput{
		Issuer:      "iss",
		AccountName: "acc",
	}
	totP, err := GenerateTotPSecretKey(inpt)
	require.NoError(t, err)

	assert.True(t, totP.Secret != "")
	assert.True(t, totP.QRImageBase64 != "")

	code, err := totp.GenerateCode(totP.Secret, time.Now())
	require.NoError(t, err)

	assert.True(t, CheckTotPCode(code, totP.Secret))
	assert.False(t, CheckTotPCode("dfasdf", totP.Secret))

	imgBytes, err := base64.StdEncoding.DecodeString(totP.QRImageBase64)
	require.NoError(t, err)

	img, err := png.DecodeConfig(bytes.NewBuffer(imgBytes))
	require.NoError(t, err)
	assert.Equal(t, img.Width, DefaultTotPQrImageWidth)
	assert.Equal(t, img.Height, DefaultTotPQrImageHeight)
}

func TestStoreTotPCodeInUser(t *testing.T) {
	usr := new(users.User)
	providedTotP := &TotP{
		Secret:        "sec123",
		QRImageBase64: "alalala",
	}

	err := StoreTotPCodeInUser(usr, providedTotP)
	require.NoError(t, err)

	assert.Equal(t, `{"secret":"sec123","qr":"alalala"}`, usr.TotP)

	actualTotP, err := GetUsersTotPCode(usr)
	require.NoError(t, err)
	assert.Equal(t, providedTotP, actualTotP)

	usr.TotP = ""
	actualTotP2, err := GetUsersTotPCode(usr)
	require.NoError(t, err)
	assert.Equal(t, "", actualTotP2.Secret)
	assert.Equal(t, "", actualTotP2.QRImageBase64)
}
