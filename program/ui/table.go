package ui

import (
	"github.com/deweysasser/olympus/terraform"
	"sort"
)

type ChangeTable struct {
	Columns []string
	Rows    []Row
}

type Row struct {
	Name     RowName
	Contents []terraform.PlanSummary
}

type RowName string

// CreateTable parses out a set of summaries and arranges it for nice display
func CreateTable(summaries []terraform.PlanSummary) *ChangeTable {
	tab := &ChangeTable{}
	table := make(map[terraform.PlanSummary]map[RowName]terraform.PlanSummary)

	rowNameMap := make(map[string]bool)

	for _, s := range summaries {
		tab.Columns = append(tab.Columns, s.Name())
		table[s] = make(map[RowName]terraform.PlanSummary)

		for _, child := range s.Children() {
			table[s][RowName(child.Name())] = child
			rowNameMap[child.Name()] = true
		}
	}

	var rowNames []RowName

	for k, _ := range rowNameMap {
		rowNames = append(rowNames, RowName(k))
	}

	sort.Slice(rowNames, func(i, j int) bool {
		return rowNames[i] < rowNames[j]
	})

	for _, rowName := range rowNames {
		row := Row{Name: rowName, Contents: make([]terraform.PlanSummary, 0)}
		for _, s := range summaries {
			row.Contents = append(row.Contents, table[s][rowName])
		}
		tab.Rows = append(tab.Rows, row)
	}

	return tab
}
