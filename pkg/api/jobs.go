package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	testgridv1 "github.com/openshift/sippy/pkg/apis/testgrid/v1"
	"github.com/openshift/sippy/pkg/testgridanalysis/testgridanalysisapi"
	"github.com/openshift/sippy/pkg/testgridanalysis/testgridconversion"
)

func jobRunStatus(result testgridanalysisapi.RawJobRunResult) string {
	if result.Succeeded {
		return "S" // Success
	}

	if !result.Failed {
		return "R" // Running
	}

	if result.SetupStatus == "Failure" {
		if len(result.FinalOperatorStates) == 0 {
			return "N" // iNfrastructure failure
		}
		return "I" // Install failure
	}
	if result.UpgradeStarted && (result.UpgradeForOperatorsStatus == "Failure" || result.UpgradeForMachineConfigPoolsStatus == "Failure") {
		return "U" // Upgrade failure
	}
	if result.OpenShiftTestsStatus == "Failure" {
		return "F" // Failure
	}
	if result.SetupStatus == "" {
		return "n" // no setup results
	}
	return "f" // unknown failure
}

func PrintJobsReport(w http.ResponseWriter, syntheticTestManager testgridconversion.SythenticTestManager, testGridJobDetails []testgridv1.JobDetails, lastUpdateTime time.Time) {
	rawJobResultOptions := testgridconversion.ProcessingOptions{
		SythenticTestManager: syntheticTestManager,
		StartDay:             0,
		NumDays:              1000,
	}
	rawJobResults, _ := rawJobResultOptions.ProcessTestGridDataIntoRawJobResults(testGridJobDetails)

	type jsonJob struct {
		Name        string   `json:"name"`
		Timestamps  []int    `json:"timestamps"`
		Results     []string `json:"results"`
		BuildIDs    []string `json:"build_ids"`
		TestGridURL string   `json:"testgrid_url"`
	}
	type jsonResponse struct {
		Jobs           []jsonJob `json:"jobs"`
		LastUpdateTime time.Time `json:"last_update_time"`
	}

	response := jsonResponse{
		LastUpdateTime: lastUpdateTime,
		Jobs:           []jsonJob{},
	}
	for _, job := range testGridJobDetails {
		results := rawJobResults.JobResults[job.Name]
		var statuses []string
		for i := range job.Timestamps {
			joburl := fmt.Sprintf("https://prow.svc.ci.openshift.org/view/gcs/%s/%s", job.Query, job.ChangeLists[i])
			statuses = append(statuses, jobRunStatus(results.JobRunResults[joburl]))
		}
		response.Jobs = append(response.Jobs, jsonJob{
			Name:        job.Name,
			Timestamps:  job.Timestamps,
			Results:     statuses,
			BuildIDs:    job.ChangeLists,
			TestGridURL: job.TestGridUrl,
		})
	}

	json.NewEncoder(w).Encode(response)
}
