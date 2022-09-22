package server

import (
	"github.com/deweysasser/olympus/git"
	tfjson "github.com/hashicorp/terraform-json"
	"time"
)

type PlanRecord struct {
	Plan  *tfjson.Plan `json:"plan"`
	Start time.Time
	End   time.Time
	Hash  git.CommitSHA
}
