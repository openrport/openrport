package chserver

import (
	"bytes"
	"encoding/base64"
	"image/png"
	"testing"
	"time"

	"github.com/realvnc-labs/rport/server/api/users"

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

	assert.True(t, CheckTotPCode(code, totP))
	assert.False(t, CheckTotPCode("dfasdf", totP))

	imgBytes, err := base64.StdEncoding.DecodeString(totP.QRImageBase64)
	require.NoError(t, err)

	img, err := png.DecodeConfig(bytes.NewBuffer(imgBytes))
	require.NoError(t, err)
	assert.Equal(t, img.Width, DefaultTotPQrImageWidth)
	assert.Equal(t, img.Height, DefaultTotPQrImageHeight)
}

func TestStoreTotPCodeInUser(t *testing.T) {
	usr := new(users.User)

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "someIss",
		AccountName: "someAcc",
	})
	require.NoError(t, err)

	providedTotP, err := NewTotP(key)
	require.NoError(t, err)

	StoreTotPCodeInUser(usr, providedTotP)

	assert.Equal(t, key.URL(), usr.TotP)

	actualTotP, err := GetUsersTotPCode(usr)
	require.NoError(t, err)
	assert.Equal(t, providedTotP, actualTotP)
}

func TestGetTotPCodeFromUser(t *testing.T) {
	testCases := []struct {
		name                     string
		providedTotPCode         string
		expectedError            string
		isFullErrorMatchExpected bool
	}{
		{
			name: "empty totp code",
		},
		{
			name:                     "invalid tot p code",
			providedTotPCode:         "sss",
			isFullErrorMatchExpected: false,
			expectedError:            "failed to convert 'sss' to TotP secret data",
		},
		{
			name:             "valid tot p code",
			providedTotPCode: "otpauth://totp/some:ap?algorithm=SHA1&digits=6&issuer=rport_with_mysql&period=30&secret=dfadfsadfas",
		},
		{
			name:                     "invalid scheme",
			providedTotPCode:         "lala://totp/some:ap?algorithm=SHA1&digits=6&issuer=some&period=30&secret=dfadfsadfas",
			expectedError:            "failed to convert 'lala://totp/some:ap?algorithm=SHA1&digits=6&issuer=some&period=30&secret=dfadfsadfas' to TotP secret data: unexpected totp key scheme lala, otpauth is expected",
			isFullErrorMatchExpected: true,
		},
		{
			name:                     "invalid type",
			providedTotPCode:         "otpauth://lala/some:ap?algorithm=SHA1&digits=6&issuer=some&period=30&secret=dfadfsadfas",
			expectedError:            "failed to convert 'otpauth://lala/some:ap?algorithm=SHA1&digits=6&issuer=some&period=30&secret=dfadfsadfas' to TotP secret data: invalid totp key type lala, totp type is expected",
			isFullErrorMatchExpected: true,
		},
		{
			name:                     "invalid algo",
			providedTotPCode:         "otpauth://totp/some:ap?algorithm=lala&digits=6&issuer=some&period=30&secret=dfadfsadfas",
			expectedError:            "failed to convert 'otpauth://totp/some:ap?algorithm=lala&digits=6&issuer=some&period=30&secret=dfadfsadfas' to TotP secret data: invalid/unsupported algorithm value lala",
			isFullErrorMatchExpected: true,
		},
		{
			name:                     "invalid digits",
			providedTotPCode:         "otpauth://totp/some:ap?algorithm=SHA1&digits=10&issuer=some&period=30&secret=dfadfsadfas",
			expectedError:            "failed to convert 'otpauth://totp/some:ap?algorithm=SHA1&digits=10&issuer=some&period=30&secret=dfadfsadfas' to TotP secret data: invalid digits value 10",
			isFullErrorMatchExpected: true,
		},
		{
			name:                     "invalid secret",
			providedTotPCode:         "otpauth://totp/some:ap?algorithm=SHA1&digits=8&issuer=some&period=30&secret=",
			expectedError:            "failed to convert 'otpauth://totp/some:ap?algorithm=SHA1&digits=8&issuer=some&period=30&secret=' to TotP secret data: empty totp secret key",
			isFullErrorMatchExpected: true,
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(testCase.name, func(t *testing.T) {
			usr := &users.User{
				TotP: tc.providedTotPCode,
			}

			actualTotP, err := GetUsersTotPCode(usr)
			if tc.expectedError != "" {
				if tc.isFullErrorMatchExpected {
					require.EqualError(t, err, tc.expectedError)
				} else {
					require.Contains(t, err.Error(), tc.expectedError)
				}
				return
			}

			require.NoError(t, err)
			actualTotPStr := actualTotP.Serialize()
			assert.Equal(t, tc.providedTotPCode, actualTotPStr)
		})
	}
}
