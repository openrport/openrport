package chserver

import (
	"bytes"
	"fmt"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

const (
	DefaultTotPQrImageWidth    = 200
	DefaultTotPQrImageHeight   = 200
	DefaultTotPQrImageFileMode = os.FileMode(0644)
)

func CheckTotPCode(code, secretKey string) bool {
	return totp.Validate(code, secretKey)
}

func GenerateTotPSecretKey(issuer, accountName, imagePath string, codeOutput io.Writer) error {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: accountName,
	})

	if err != nil {
		return err
	}

	if imagePath != "" {
		err = generateImage(imagePath, key)
		if err != nil {
			return err
		}
	}

	_, err = codeOutput.Write([]byte(key.Secret()))

	return err
}

func generateImage(imagePath string, key *otp.Key) error {
	var buf bytes.Buffer
	img, err := key.Image(DefaultTotPQrImageWidth, DefaultTotPQrImageHeight)
	if err != nil {
		return err
	}

	ext := filepath.Ext(imagePath)
	switch ext {
	case ".png":
		err = png.Encode(&buf, img)
		if err != nil {
			return err
		}
	case ".jpg":
		err = jpeg.Encode(&buf, img, nil)
		if err != nil {
			return err
		}

	default:
		return fmt.Errorf("unsupported image format %s", ext)
	}

	err = ioutil.WriteFile(imagePath, buf.Bytes(), DefaultTotPQrImageFileMode)
	if err != nil {
		return err
	}

	return nil
}
