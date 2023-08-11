package transformers

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/realvnc-labs/rport/plus/capabilities/alerting/entities/measures"
)

func TestShouldTransformJSONToMountPoints(t *testing.T) {
	testCases := []struct {
		Name                string
		MountPointsJSON     string
		ExpectedMountPoints []measures.MountPoint
		Err                 error
	}{
		{
			Name: "Valid JSON",
			MountPointsJSON: `{
				"free_b.disk1": 100,
				"total_b.disk1": 500,
				"free_b.disk2": 200,
				"total_b.disk2": 1000
			}`,
			ExpectedMountPoints: []measures.MountPoint{
				{
					Name:        "disk1",
					FreeBytes:   100,
					TotalBytes:  500,
					FreePercent: 100 - ((100.00 / 500) * 100),
					UsedPercent: (100.00 / 500) * 100,
				},
				{
					Name:        "disk2",
					FreeBytes:   200,
					TotalBytes:  1000,
					FreePercent: 100 - ((200.00 / 1000) * 100),
					UsedPercent: (200.00 / 1000) * 100,
				},
			},
			Err: nil,
		},
		{
			Name: "Invalid JSON",
			MountPointsJSON: `{
				"free_b.disk1": 100
				"total_b.disk1": 500,
			}`,
			ExpectedMountPoints: nil,
			Err:                 fmt.Errorf("msg:\"invalid character '\"' after object key:value pair"),
		},
		{
			Name: "Invalid Key Format",
			MountPointsJSON: `{
				"free_b.disk1": 100,
				"total_b.disk1": 500,
				"invalid_key": 200,
				"total_b.disk2": 1000
			}`,
			ExpectedMountPoints: nil,
			Err:                 fmt.Errorf("unable to process mount point info item: invalid_key"),
		},
		{
			Name: "Multiple Values for Same Mount Point",
			MountPointsJSON: `{
				"free_b.disk1": 100,
				"total_b.disk1": 500,
				"free_b.disk1": 200,
				"total_b.disk1": 1000
			}`,
			ExpectedMountPoints: []measures.MountPoint{
				{
					Name:        "disk1",
					FreeBytes:   200,
					TotalBytes:  1000,
					FreePercent: 100 - ((200.00 / 1000) * 100),
					UsedPercent: (200.00 / 1000) * 100,
				},
			},
			Err: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			mountPoints, err := TransformMountPointsJSONToMountPoints(tc.MountPointsJSON)

			if tc.Err != nil {
				assert.Contains(t, tc.Err.Error(), err.Error())
			} else {
				assert.Equal(t, tc.ExpectedMountPoints, mountPoints)
			}
		})
	}
}
