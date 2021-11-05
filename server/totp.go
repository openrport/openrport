package chserver

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/cloudradar-monitoring/rport/server/api/users"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"image/png"
)

const (
	DefaultTotPQrImageWidth  = 200
	DefaultTotPQrImageHeight = 200
)

type TotPInput struct {
	Issuer      string
	AccountName string
}

type TotP struct {
	Secret        string `json:"secret"`
	QRImageBase64 string `json:"qr"`
}

func GetUsersTotPCode(usr *users.User) (*TotP, error) {
	totP := new(TotP)
	if usr.TotP == "" {
		return totP, nil
	}

	err := json.Unmarshal([]byte(usr.TotP), totP)

	if err != nil {
		return nil, fmt.Errorf("failed to convert '%s' to TotP secret data", usr.TotP)
	}

	return totP, nil
}

func StoreTotPCodeInUser(usr *users.User, totP *TotP) error {
	totPBytes, err := json.Marshal(totP)

	if err != nil {
		return fmt.Errorf("failed to generate totP secret data: %v", err)
	}

	usr.TotP = string(totPBytes)

	return nil
}

func CheckTotPCode(code, secretKey string) bool {
	return totp.Validate(code, secretKey)
}

func GenerateTotPSecretKey(inpt *TotPInput) (*TotP, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      inpt.Issuer,
		AccountName: inpt.AccountName,
	})

	if err != nil {
		return nil, err
	}

	imgBase64, err := generateImage(key)
	if err != nil {
		return nil, err
	}

	totP := &TotP{
		Secret:        key.Secret(),
		QRImageBase64: imgBase64,
	}

	return totP, err
}

func generateImage(key *otp.Key) (imgBase64 string, err error) {
	var buf bytes.Buffer
	img, err := key.Image(DefaultTotPQrImageWidth, DefaultTotPQrImageHeight)
	if err != nil {
		return "", err
	}

	err = png.Encode(&buf, img)
	if err != nil {
		return "", err
	}

	imgBase64 = base64.StdEncoding.EncodeToString(buf.Bytes())

	return imgBase64, nil
}
