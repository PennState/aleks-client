package main

import (
	log "github.com/sirupsen/logrus"

	"github.com/selesy/aleks-client/pkg/aleks"
)

func main() {
	client, err := aleks.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	pr, err := client.GetPlacementReportFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	log.Info("Placement report: ", pr)
}
