package encoding

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding/charmap"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

func TestToUTF8(t *testing.T) {
	// given
	textUTF8 := "Configuración IP de Windows"
	enc := charmap.ISO8859_1
	en := enc.NewEncoder()
	nonUTF8, err := en.Bytes([]byte(textUTF8))
	require.NoError(t, err)
	require.NotEqual(t, string(nonUTF8), textUTF8)
	testLog := chshare.NewLogger("server", chshare.LogOutput{File: os.Stdout}, chshare.LogLevelDebug)

	// when
	gotText, err := ToUTF8(testLog, nonUTF8)

	// then
	require.NoError(t, err)
	assert.Equal(t, gotText, textUTF8)
}
