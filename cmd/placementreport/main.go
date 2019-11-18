package main

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/PennState/aleks-client/pkg/aleks"
)

func main() {
	client, err := aleks.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	start := time.Now()
	pr, errs := client.GetPlacementReportFromEnv()
	end := time.Now()
	log.Info("Placement report: ", pr)
	log.Info("Errors: ", errs)
	log.Info("Start time: ", start)
	log.Info("End time: ", end)
	log.Info("Placement record count: ", len(pr))
	log.Info("Error count: ", len(errs))
}
