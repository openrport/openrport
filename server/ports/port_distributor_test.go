package ports

import (
	"testing"

	mapset "github.com/deckarep/golang-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudradar-monitoring/rport/share/models"
)

func TestPortDistributor(t *testing.T) {

	for _, protocol := range []string{models.ProtocolTCP, models.ProtocolUDP, models.ProtocolTCPUDP} {
		t.Run(protocol, func(t *testing.T) {
			pd := NewPortDistributorForTests(
				mapset.NewSetFromSlice([]interface{}{1, 2, 3, 4, 5}),
				mapset.NewSetFromSlice([]interface{}{2, 3, 4, 5}),
				mapset.NewSetFromSlice([]interface{}{2, 3, 4, 5}),
			)

			assert.Equal(t, true, pd.IsPortBusy(protocol, 1))
			assert.Equal(t, false, pd.IsPortBusy(protocol, 2))

			port, err := pd.GetRandomPort(protocol)
			require.NoError(t, err)

			assert.Equal(t, true, pd.IsPortBusy(protocol, port))
		})
	}
}
