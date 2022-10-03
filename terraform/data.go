package terraform

type SummaryData struct {
	Name             string        `json:"name"`
	Changes          Changes       `json:"changes"`
	Children         []PlanSummary `json:"children,omitempty" `
	ChangedResources string        `json:"changed-resources,omitempty"`
}

type Changes struct {
	Added   int `json:"added"`
	Updated int `json:"updated"`
	Deleted int `json:"deleted"`
}

func (c Changes) HasAny() bool {
	return c.Added+c.Updated+c.Deleted > 0
}

func (c Changes) Highest() string {
	switch {
	case c.Deleted > 0:
		return "deleted"
	case c.Updated > 0:
		return "updated"
	case c.Added > 0:
		return "added"
	default:
		return "none"
	}
}
