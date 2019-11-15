package aleks

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlacementReportRequestClasscodeValidation(t *testing.T) {
	tests := []struct {
		Name   string
		Value  string
		Errant bool
	}{
		{"Valid classcode", "ABCDE-FGHIJ", false},
		{"Short prefix", "", true},
		{"Long prefix", "", true},
		{"Short suffix", "", true},
		{"Long suffix", "", true},
		{"No dash", "", true},
		{"Invalid character", "", true},
	}
	for idx := range tests {
		test := tests[idx]
		t.Run(test.Name, func(t *testing.T) {
			errs := validateClassCodes([]string{test.Value})
			if test.Errant {
				require.Len(t, errs, 1)
				assert.Equal(t, classcodeValidationErrorMessage+test.Value, errs[0].Error())
			}
		})
	}
}

func TestPlacementReportRequestDateValidation(t *testing.T) {
	invalidErr := &time.ParseError{
		Value:   "10/11/2019",
		Message: " as \"2006-01-02\": cannot parse \"1/2019\" as \"2006\"",
	}
	tests := []struct {
		Name     string
		Value    string
		Expected []error
	}{
		{"Valid request date format", "2019-10-11", []error{}},
		{"Invalid request date format", "10/11/2019", []error{invalidErr}},
	}
	for idx := range tests {
		test := tests[idx]
		t.Run(test.Name, func(t *testing.T) {
			errs := validateRequestDate(test.Value)
			assert.ObjectsAreEqual(test.Expected, errs)
		})
	}
}
