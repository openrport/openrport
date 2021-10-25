package chserver

import (
	"bytes"
	"image"
	"os"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCodeGenerationAndValidation(t *testing.T) {
	var buf bytes.Buffer
	err := GenerateTotPSecretKey("iss", "acc", "", &buf)
	require.NoError(t, err)

	assert.True(t, buf.String() != "")

	code, err := totp.GenerateCode(buf.String(), time.Now())
	require.NoError(t, err)

	assert.True(t, CheckTotPCode(code, buf.String()))
	assert.False(t, CheckTotPCode("12345", buf.String()))
}

func TestImageGeneration(t *testing.T) {
	fileNames := []string{"TestImageGeneration.png", "TestImageGeneration.jpg"}
	defer func() {
		for _, fileName := range fileNames {
			os.Remove(fileName)
		}
	}()

	for _, fileName := range fileNames {
		fn := fileName
		t.Run(fn, func(t *testing.T) {
			var buf bytes.Buffer
			err := GenerateTotPSecretKey("iss", "acc", fn, &buf)
			require.NoError(t, err)

			fileStat, err := os.Stat(fn)
			require.NoError(t, err)
			assert.Equal(t, fn, fileStat.Name())
			assert.False(t, fileStat.IsDir())
			assert.True(t, fileStat.Size() > 0)
			assert.Equal(t, DefaultTotPQrImageFileMode, fileStat.Mode())

			f, err := os.Open(fn)
			require.NoError(t, err)
			defer f.Close()

			img, _, err := image.DecodeConfig(f)
			require.NoError(t, err)
			assert.Equal(t, img.Width, DefaultTotPQrImageWidth)
			assert.Equal(t, img.Height, DefaultTotPQrImageHeight)
		})
	}

	var buf bytes.Buffer
	err := GenerateTotPSecretKey("iss", "acc", "TestImageGeneration.tiff", &buf)
	require.EqualError(t, err, "unsupported image format .tiff")
}
