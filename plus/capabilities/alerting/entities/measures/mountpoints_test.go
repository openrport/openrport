package measures

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldCalculateFreePercent(t *testing.T) {
	tests := []struct {
		name          string
		mp            MountPoint
		expectedValue float64
		expectedError error
	}{
		{
			name: "normal free percent",
			mp: MountPoint{
				Name:       "root",
				FreeBytes:  1,
				TotalBytes: 10,
			},
			expectedValue: 90.0,
		},
		{
			name: "incorrect free",
			mp: MountPoint{
				Name:       "root",
				FreeBytes:  20,
				TotalBytes: 10,
			},
			expectedValue: -100.0,
		},
		{
			name: "zero free",
			mp: MountPoint{
				Name:       "root",
				FreeBytes:  0,
				TotalBytes: 10,
			},
			expectedValue: 100.0,
		},
		{
			name: "zero total",
			mp: MountPoint{
				Name:       "root",
				FreeBytes:  10,
				TotalBytes: 0,
			},
			expectedValue: 0.0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			percent := tc.mp.CalcFreePercent()
			assert.Equal(t, tc.expectedValue, percent)
		})
	}
}

func TestShouldCalculateUsedPercent(t *testing.T) {
	tests := []struct {
		name          string
		mp            MountPoint
		expectedValue float64
		expectedError error
	}{
		{
			name: "normal used percent",
			mp: MountPoint{
				Name:       "root",
				FreeBytes:  1,
				TotalBytes: 10,
			},
			expectedValue: 10.0,
		},
		{
			name: "incorrect free",
			mp: MountPoint{
				Name:       "root",
				FreeBytes:  20,
				TotalBytes: 10,
			},
			expectedValue: 200.0,
		},
		{
			name: "zero free",
			mp: MountPoint{
				Name:       "root",
				FreeBytes:  0,
				TotalBytes: 10,
			},
			expectedValue: 0.0,
		},
		{
			name: "zero total",
			mp: MountPoint{
				Name:       "root",
				FreeBytes:  10,
				TotalBytes: 0,
			},
			expectedValue: 100.0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			percent := tc.mp.CalcUsedPercent()
			assert.Equal(t, tc.expectedValue, percent)
		})
	}
}
