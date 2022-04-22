package query

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequestedFields(t *testing.T) {
	fields := []FieldsOption{
		{
			Resource: "abc",
			Fields:   []string{"f1", "f2"},
		},
		{
			Resource: "def",
			Fields:   []string{"f4", "f3"},
		},
	}

	result := RequestedFields(fields, "abc")

	expected := map[string]bool{
		"f1": true,
		"f2": true,
	}
	assert.Equal(t, expected, result)
}
