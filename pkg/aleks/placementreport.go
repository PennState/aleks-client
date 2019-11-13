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
	placementReportMethod            = "getPlacementReport"
	placementReportRequestDateFormat = "2006-01-02"
	placementReportHeaderColumn00    = "Name"
	placementReportHeaderColumn01    = "Student Id"
	placementReportHeaderColumn02    = "Email"
	placementReportHeaderColumn03    = "Last login"
	placementReportHeaderColumn04    = "Placement Assessment Number"
	placementReportHeaderColumn05    = "Total Number of Placements Taken"
	placementReportHeaderColumn06    = "Start Date"
	placementReportHeaderColumn07    = "Start Time"
	placementReportHeaderColumn08    = "End Date"
	placementReportHeaderColumn09    = "End Time"
	placementReportHeaderColumn10    = "Proctored Assessment"
	placementReportHeaderColumn11    = "Time in Placement (in hours)"
	placementReportHeaderColumn12    = "Placement Results %"
	placementRecordFieldCount        = 13
	placementRecordDateFormat        = "01/02/2006"
	placementRecordTimestampFormat   = "01/02/2006 03:04 PM"
)

type PlacementReport []PlacementRecord

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
		params := map[string]string{
			"username":             c.username,
			"password":             c.password,
			"from_completion_date": from,
			"to_completion_date":   to,
			"class_code":           code,
		}
		go func() {
			pr, err := c.getPlacementReportForClassCode(xc, params)
			r <- result{pr, err}
		}()
	}

	// Gather
	pr := PlacementReport{}
	errs := []error{}
	for i := 0; i < cap(r); i++ {
		res := <-r
		if res.Errors != nil && len(res.Errors) > 0 {
			errs = append(errs, res.Errors...)
		}
		if res.PlacementReport != nil && len(res.PlacementReport) > 0 {
			pr = append(pr, res.PlacementReport...)
		}

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
	err := envconfig.Process(aleksEnvconfigPrefix, &cfg)
	if err != nil {
		return nil, err
	}
	log.Info("Placement report config: ", cfg)
	return c.GetPlacementReport(cfg.From, cfg.To, cfg.ClassCodes...)
}

func (c *Client) getPlacementReportForClassCode(xc *xmlrpc.Client, params map[string]string) (PlacementReport, []error) {
	rep := PlacementReport{}
	errs := []error{}
	// TODO: iterate over page numbers
	params["page_num"] = "1"
	data := ""
	err := xc.Call(placementReportMethod, params, &data)
	if err != nil {
		return nil, append(errs, err)
	}
	data = strings.Trim(data, " 	\n")

	rdr := csv.NewReader(strings.NewReader(data))
	rdr.FieldsPerRecord = placementRecordFieldCount
	rdr.ReuseRecord = true

	first := true
	for {
		rec, err := rdr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if first {
			errs = append(errs, checkHeaders(rec)...)
			first = false
			continue
		}
		r, e := newPlacementRecord(rec)
		errs = append(errs, e...)
		rep = append(rep, r)
	}

	// for idx, line := range strings.Split(data, "\n") {
	// 	if idx == 0 {
	// 		err = checkHeader(line)
	// 		if err != nil {
	// 			errs = append(errs, err)
	// 		}
	// 		continue
	// 	}
	// 	rec, err := newPlacementRecord(line)
	// 	if err != nil {
	// 		errs = append(errs, err)
	// 		continue
	// 	}
	// 	rep = append(rep, rec)
	// }
	return rep, errs
}

func checkHeaders(record []string) []error {
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
	StudentId                    string
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
	log.Info("Record: ", rec)
	errs := []error{}
	// TODO: replace this with functions that accumulate the errors
	lastLogin, err := time.Parse(placementRecordDateFormat, rec[3])
	if err != nil {
		errs = append(errs, err)
	}
	placementAssessmentNumber, err := strconv.ParseInt(rec[4], 10, 32)
	if err != nil {
		errs = append(errs, err)
	}
	totalNumberOfPlacementsTaken, err := strconv.ParseInt(rec[5], 10, 32)
	if err != nil {
		errs = append(errs, err)
	}
	startTime, err := time.Parse(placementRecordTimestampFormat, rec[6]+" "+rec[7])
	if err != nil {
		errs = append(errs, err)
	}
	endTime, err := time.Parse(placementRecordTimestampFormat, rec[8]+" "+rec[9])
	if err != nil {
		errs = append(errs, err)
	}
	hoursInPlacement, err := strconv.ParseFloat(rec[11], 64)
	if err != nil {
		errs = append(errs, err)
	}
	// FIXME: strip the % before parsing
	placementResults, err := strconv.ParseFloat(rec[12], 64)
	if err != nil {
		errs = append(errs, err)
	}
	return PlacementRecord{
		Name:                         rec[0],
		StudentId:                    rec[1],
		Email:                        rec[2],
		LastLogin:                    lastLogin,
		PlacementAssessmentNumber:    int(placementAssessmentNumber),
		TotalNumberOfPlacementsTaken: int(totalNumberOfPlacementsTaken),
		StartTime:                    startTime,
		EndTime:                      endTime,
		ProctoredAssessment:          rec[10],
		HoursInPlacement:             hoursInPlacement,
		PlacementResults:             placementResults,
	}, errs
}
