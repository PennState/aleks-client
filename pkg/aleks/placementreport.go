/*
Copyright 2019 The Pennsylvania State University

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package aleks

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/kolo/xmlrpc"
	log "github.com/sirupsen/logrus"
)

const (
	placementReportMethod                 = "getPlacementReport"
	placementReportRequestDateFormat      = "2006-01-02"
	placementReportRequestClasscodeFormat = "[A-Z]{5}-[A-Z]{5}"
	placementReportHeaderColumn00         = "Name"
	placementReportHeaderColumn01         = "Student Id"
	placementReportHeaderColumn02         = "Email"
	placementReportHeaderColumn03         = "Last login"
	placementReportHeaderColumn04         = "Placement Assessment Number"
	placementReportHeaderColumn05         = "Total Number of Placements Taken"
	placementReportHeaderColumn06         = "Start Date"
	placementReportHeaderColumn07         = "Start Time"
	placementReportHeaderColumn08         = "End Date"
	placementReportHeaderColumn09         = "End Time"
	placementReportHeaderColumn10         = "Proctored Assessment"
	placementReportHeaderColumn11         = "Time in Placement (in hours)"
	placementReportHeaderColumn12         = "Placement Results %"
	placementReportEndMarker              = "No records found"
	placementRecordFieldCount             = 13
	placementRecordDateFormat             = "01/02/2006"
	placementRecordTimestampFormat        = "01/02/2006 03:04 PM"
)

const (
	classcodeValidationErrorMessage = "Class code does not match required format: "
)

// PlacementReport contains the PlacementRecords returned (if any) by the
// GetPlacementReport and GetPlacementReportFromEnv methods.
type PlacementReport []PlacementRecord

// GetPlacementReport calls the Aleks XML-RPC method of the same name for
// one or more class-codes and returns the results as a list of
// PlacementRecords.  A collection of errors that occurred during this
// process is also collected and returned to the caller.  Note that it is
// possible for both PlacementRecords and errors to be returned from the
// same call as valid PlacementRecords are not discarded due to errors
// in other records.
//
// This method uses an individual thread to retrieve the data for each
// class-code and collects the results in a single PlacementReport to
// reduce the time it takes to retrieve large data sets.  Requesting
// large numbers of class-codes will therefore result in a large number
// of threads.
func (c *Client) GetPlacementReport(from, to string, classcodes ...string) (PlacementReport, []error) {
	pr := PlacementReport{}
	errs := []error{}

	errs = append(errs, validateRequestDate(from)...)
	errs = append(errs, validateRequestDate(to)...)
	errs = append(errs, validateClasscodes(classcodes)...)
	if len(errs) > 0 {
		return pr, errs
	}

	type result struct {
		PlacementReport PlacementReport
		Errors          []error
	}
	r := make(chan result, len(classcodes))

	// Scatter
	for _, code := range classcodes {
		xc, err := xmlrpc.NewClient(c.url, c.trans)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		params := map[string]string{
			"username":             c.username,
			"password":             c.password,
			"from_completion_date": from,
			"to_completion_date":   to,
			"class_code":           code,
		}
		go func() {
			pr, err := getPlacementReportForClasscode(xc, params)
			r <- result{pr, err}
		}()
	}

	// Gather
	for i := 0; i < cap(r); i++ {
		res := <-r
		errs = append(errs, res.Errors...)
		pr = append(pr, res.PlacementReport...)
	}
	return pr, errs
}

type placementReportEnvConfig struct {
	From       string   `envconfig:"FROM_COMPLETION_DATE" required:"true"`
	To         string   `envconfig:"TO_COMPLETION_DATE" required:"true"`
	Classcodes []string `required:"true"`
}

// GetPlacementReportFromEnv returns PlacementRecords and errors as
// described by the documentation for GetPlacementReport but retrieves
// its configuration from environment variables as follows:
//
//   - ALEKS_FROM_COMPLETION_DATE (Required - YYYY-MM-DD)
//   - ALEKS_TO_COMPLETION_DATE   (Required - YYYY-MM-DD)
//   - ALEKS_CLASSCODES           (Required - One or more class-codes
//                                 with the format AAAAA-AAAAA in a comma
//                                 separated string)
func (c *Client) GetPlacementReportFromEnv() (PlacementReport, []error) {
	cfg := placementReportEnvConfig{}
	err := envconfig.Process(AleksEnvconfigPrefix, &cfg)
	if err != nil {
		return nil, []error{err}
	}
	return c.GetPlacementReport(cfg.From, cfg.To, cfg.Classcodes...)
}

func getPlacementReportForClasscode(xc *xmlrpc.Client, params map[string]string) (PlacementReport, []error) {
	rep := PlacementReport{}
	errs := []error{}
	for page := 1; true; page++ {
		params["page_num"] = strconv.FormatInt(int64(page), 10)
		data := ""
		err := xc.Call(placementReportMethod, params, &data)
		if err != nil {
			return nil, append(errs, err)
		}

		data = strings.Trim(data, " 	\n")
		if data == placementReportEndMarker {
			break
		}
		log.Debug("Page data: ", data)

		r, e := getPlacementRecordsForPage(data)
		rep = append(rep, r...)
		errs = append(errs, e...)
	}
	return rep, errs
}

func getPlacementRecordsForPage(data string) (PlacementReport, []error) {
	rdr := csv.NewReader(strings.NewReader(data))
	rdr.FieldsPerRecord = placementRecordFieldCount
	rdr.ReuseRecord = true

	rep := PlacementReport{}
	errs := []error{}
	for first := true; true; first = false {
		rec, err := rdr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if first {
			errs = append(errs, validateHeaders(rec)...)
			continue
		}
		r, e := newPlacementRecord(rec)
		log.Debug("Placement record: ", r)
		errs = append(errs, e...)
		rep = append(rep, r)
	}
	return rep, errs
}

func validateClasscodes(classcodes []string) []error {
	errs := []error{}
	re, err := regexp.Compile(placementReportRequestClasscodeFormat)
	if err != nil {
		errs = append(errs, err)
	}
	for _, classcode := range classcodes {
		if !re.Match([]byte(classcode)) {
			errs = append(errs, errors.New(classcodeValidationErrorMessage+classcode))
		}
	}
	return errs
}

func validateHeaders(record []string) []error {
	errs := []error{}
	for idx, hdr := range record {
		exp := expectedHeaders()[idx]
		if hdr != exp {
			msg := fmt.Sprintf("Unexpected header column title (%d) - expected: %s, actual: %s", idx, hdr, exp)
			errs = append(errs, errors.New(msg))
		}
	}
	return errs
}

func validateRequestDate(value string) []error {
	errs := []error{}
	_, err := time.Parse(placementReportRequestDateFormat, value)
	if err != nil {
		errs = append(errs, err)
	}
	return errs
}

func expectedHeaders() []string {
	return []string{
		placementReportHeaderColumn00,
		placementReportHeaderColumn01,
		placementReportHeaderColumn02,
		placementReportHeaderColumn03,
		placementReportHeaderColumn04,
		placementReportHeaderColumn05,
		placementReportHeaderColumn06,
		placementReportHeaderColumn07,
		placementReportHeaderColumn08,
		placementReportHeaderColumn09,
		placementReportHeaderColumn10,
		placementReportHeaderColumn11,
		placementReportHeaderColumn12,
	}
}

// PlacementRecord provides the results of an individual placement exam.
// The Aleks format returns 13 strings formatted as a CSV record.  The
// 11 fields below are in the same order and (almost) have the same names
// as the columns in this CSV report - the "Start Date"/"Start Time" and
// "End Date"/"End Time" columns are combined into a single field below.
// In addition, all string columns are validated and converted to their
// appropriate types.
type PlacementRecord struct {
	Name                         string
	StudentID                    string
	Email                        string
	LastLogin                    time.Time
	PlacementAssessmentNumber    int
	TotalNumberOfPlacementsTaken int
	StartTime                    time.Time
	EndTime                      time.Time
	ProctoredAssessment          string
	HoursInPlacement             float64
	PlacementResults             float64
}

func newPlacementRecord(rec []string) (PlacementRecord, []error) {
	log.Debug(" CSV record: ", rec)
	errs := []error{}
	lastLogin, errs := parseDate(rec[3], errs)
	placementAssessmentNumber, errs := parseInt(rec[4], errs)
	totalNumberOfPlacementsTaken, errs := parseInt(rec[5], errs)
	startTime, errs := parseTime(rec[6]+" "+rec[7], errs)
	endTime, errs := parseTime(rec[8]+" "+rec[9], errs)
	hoursInPlacement, errs := parseFloat(rec[11], errs)
	placementResults, errs := parseFloat(rec[12], errs)
	return PlacementRecord{
		Name:                         rec[0],
		StudentID:                    rec[1],
		Email:                        rec[2],
		LastLogin:                    lastLogin,
		PlacementAssessmentNumber:    placementAssessmentNumber,
		TotalNumberOfPlacementsTaken: totalNumberOfPlacementsTaken,
		StartTime:                    startTime,
		EndTime:                      endTime,
		ProctoredAssessment:          rec[10],
		HoursInPlacement:             hoursInPlacement,
		PlacementResults:             placementResults,
	}, errs
}

func parseDate(value string, errs []error) (time.Time, []error) {
	d, err := time.Parse(placementRecordDateFormat, value)
	if err != nil {
		errs = append(errs, err)
	}
	return d, errs
}

func parseInt(value string, errs []error) (int, []error) {
	i, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		errs = append(errs, err)
	}
	return int(i), errs
}

func parseFloat(value string, errs []error) (float64, []error) {
	value = strings.ReplaceAll(value, "%", "")
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		errs = append(errs, err)
	}
	return f, errs
}

func parseTime(value string, errs []error) (time.Time, []error) {
	t, err := time.Parse(placementRecordTimestampFormat, value)
	if err != nil {
		errs = append(errs, err)
	}
	return t, errs
}
