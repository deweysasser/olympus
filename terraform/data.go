package terraform

type SummaryData struct {
	Name             string        `json:"name"`
	Changes          Changes       `json:"changes"`
	Children         []PlanSummary `json:"children,omitempty" `
	ChangedResources string        `json:"changed-resources,omitempty"`
}

type Changes struct {
	ResourcesAdded   int `json:"added"`
	ResourcesUpdated int `json:"updated"`
	ResourcesDeleted int `json:"deleted"`
}

func (c Changes) HasAny() bool {
	return c.ResourcesAdded+c.ResourcesUpdated+c.ResourcesDeleted > 0
}

func (c Changes) Highest() string {
	switch {
	case c.ResourcesDeleted > 0:
		return "deleted"
	case c.ResourcesUpdated > 0:
		return "updated"
	case c.ResourcesAdded > 0:
		return "added"
	default:
		return "none"
	}
}
