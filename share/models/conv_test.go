package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStrToBool(t *testing.T) {
	testCases := []struct {
		InputStr string
		WantRes  bool
	}{
		{
			InputStr: "1",
			WantRes:  true,
		},
		{
			InputStr: "",
			WantRes:  false,
		},
		{
			InputStr: "false",
			WantRes:  false,
		},
		{
			InputStr: "0",
			WantRes:  false,
		},
		{
			InputStr: "true",
			WantRes:  true,
		},
		{
			InputStr: "lala",
			WantRes:  true,
		},
		{
			InputStr: "-1",
			WantRes:  true,
		},
	}

	for _, testCase := range testCases {
		assert.Equal(t, testCase.WantRes, StrToBool(testCase.InputStr))
	}
}
