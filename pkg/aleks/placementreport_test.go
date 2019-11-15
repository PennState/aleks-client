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
			errs := validateClasscodes([]string{test.Value})
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

func TestPlacementReport(t *testing.T) {
	//nolint:lll
	data := `
"Name","Student Id","Email","Last login","Placement Assessment Number","Total Number of Placements Taken","Start Date","Start Time","End Date","End Time","Proctored Assessment","Time in Placement (in hours)","Placement Results %"
"Doe, John","912345678","JQD5678@PSU.EDU","03/06/2016","1","1","03/06/2016","01:42 PM","03/06/2016","03:23 PM","No/Complete","1.7","62%"
"Doe, Jane","923456789","JXD6789@PSU.EDU","03/03/2016","1","2","03/03/2016","07:19 PM","03/03/2016","09:30 PM","No/Complete","2.2","81%"
`
	exp := PlacementReport{
		PlacementRecord{
			Name:                         "Doe, John",
			StudentID:                    "912345678",
			Email:                        "JQD5678@PSU.EDU",
			LastLogin:                    time.Date(2016, time.March, 6, 0, 0, 0, 0, time.UTC),
			PlacementAssessmentNumber:    1,
			TotalNumberOfPlacementsTaken: 1,
			StartTime:                    time.Date(2016, time.March, 6, 13, 42, 0, 0, time.UTC),
			EndTime:                      time.Date(2016, time.March, 6, 15, 23, 0, 0, time.UTC),
			ProctoredAssessment:          "No/Complete",
			HoursInPlacement:             1.7,
			PlacementResults:             62,
		},
		PlacementRecord{
			Name:                         "Doe, Jane",
			StudentID:                    "923456789",
			Email:                        "JXD6789@PSU.EDU",
			LastLogin:                    time.Date(2016, time.March, 3, 0, 0, 0, 0, time.UTC),
			PlacementAssessmentNumber:    1,
			TotalNumberOfPlacementsTaken: 2,
			StartTime:                    time.Date(2016, time.March, 3, 19, 19, 0, 0, time.UTC),
			EndTime:                      time.Date(2016, time.March, 3, 21, 30, 0, 0, time.UTC),
			ProctoredAssessment:          "No/Complete",
			HoursInPlacement:             2.2,
			PlacementResults:             81,
		},
	}
	pr, errs := getPlacementRecordsForPage(data)
	require.Len(t, errs, 0)
	require.Len(t, pr, 2)
	assert.Equal(t, exp, pr)
}
