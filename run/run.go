package run

import (
	"github.com/deweysasser/olympus/git"
	"github.com/deweysasser/olympus/terraform"
	tfjson "github.com/hashicorp/terraform-json"
	"time"
)

type PlanRecord struct {
	Plan      *tfjson.Plan        `json:"plan,omitempty"`
	Start     time.Time           `json:"start-time"`
	End       time.Time           `json:"end-time"`
	CommitSHA git.SHA256          `json:"commit-sha"`
	Repo      git.Repo            `json:"repo"`
	Branch    git.Branch          `json:"branch"`
	Workspace terraform.Workspace `json:"workspace"`
	Command   string              `json:"command"`
	Output    string              `json:"output,omitempty"`
	Succeeded bool                `json:"success"`
}

type SummaryInfo struct {
	Name             string            `json:"name"`
	Changes          terraform.Changes `json:"changes"`
	UpToDate         bool              `json:"up_to_date"`
	ChangedResources string            `json:"changed_resources,omitempty"`
}

// Summary is a run record to which summary information has been added
type Summary struct {
	PlanRecord
	SummaryInfo
}

type Group struct {
	SummaryInfo
	Oldest time.Time `json:"oldest"`
	Newest time.Time `json:"newest"`

	Records []Summary `json:"records"`
}

type Set struct {
	SummaryInfo

	// TODO: do we need this at all levels, or just here?
	Branch    git.Branch          `json:"branch"`
	Workspace terraform.Workspace `json:"workspace"`
	Oldest    time.Time           `json:"oldest"`
	Newest    time.Time           `json:"newest"`

	Records []Group `json:"records"`
}
