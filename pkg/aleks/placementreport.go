package aleks

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
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
	placementReportRequestClassCodeFormat = "[A-Z]{5}-[A-Z]{5}"
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

// PlacementReport contains the PlacementRecords returned (if any) by the
// GetPlacementReport method.
type PlacementReport []PlacementRecord

//
func (c *Client) GetPlacementReport(from, to string, classcodes ...string) (PlacementReport, error) {
	type result struct {
		PlacementReport PlacementReport
		Errors          []error
	}
	r := make(chan result, len(classcodes))
	// Scatter
	for _, code := range classcodes {
		xc, err := xmlrpc.NewClient(c.url, c.trans)
		if err != nil {
			return nil, err
		}
		// TODO: validate from and to are formatted correctly
		// TODO: validate classcodes are formatted correctly
		params := map[string]string{
			"username":             c.username,
			"password":             c.password,
			"from_completion_date": from,
			"to_completion_date":   to,
			"class_code":           code,
		}
		go func() {
			pr, err := getPlacementReportForClassCode(xc, params)
			r <- result{pr, err}
		}()
	}

	// Gather
	pr := PlacementReport{}
	errs := []error{}
	for i := 0; i < cap(r); i++ {
		res := <-r
		errs = append(errs, res.Errors...)
		pr = append(pr, res.PlacementReport...)
	}
	log.Info("Errors: ", errs)
	return pr, nil
}

type placementReportEnvConfig struct {
	From       string   `envconfig:"FROM_COMPLETION_DATE" required:"true"`
	To         string   `envconfig:"TO_COMPLETION_DATE" required:"true"`
	ClassCodes []string `split_words:"true" required:"true"`
}

func (c *Client) GetPlacementReportFromEnv() (PlacementReport, error) {
	cfg := placementReportEnvConfig{}
	err := envconfig.Process(AleksEnvconfigPrefix, &cfg)
	if err != nil {
		return nil, err
	}
	log.Info("Placement report config: ", cfg)
	return c.GetPlacementReport(cfg.From, cfg.To, cfg.ClassCodes...)
}

func getPlacementReportForClassCode(xc *xmlrpc.Client, params map[string]string) (PlacementReport, []error) {
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
