package utils

import (
	"os"

	"github.com/olekukonko/tablewriter"
)

type TableConfigs struct {
	Seperator   string
	TableHeader []string
	TableBorder tablewriter.Border
}

// LogTable logs table with given configs and data
func LogTable(configs TableConfigs, data [][]string) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(configs.TableHeader)
	table.SetBorders(configs.TableBorder)
	table.SetCenterSeparator(configs.Seperator)
	table.AppendBulk(data)
	table.Render()
}

// CreateTableConfigs creates and returns an object of the type TableConfigs
func CreateTableConfigs(border tablewriter.Border, header []string, seperator string) TableConfigs {
	return TableConfigs{
		TableBorder: border,
		TableHeader: header,
		Seperator:   seperator,
	}
}

// CreateBorder creates and returns an object of the type tablewriter.Border
func CreateBorder(left bool, right bool, top bool, bottom bool) tablewriter.Border {
	return tablewriter.Border{
		Left:   left,
		Right:  right,
		Top:    top,
		Bottom: bottom,
	}
}
