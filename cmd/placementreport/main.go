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
