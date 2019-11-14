package aleks

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

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
