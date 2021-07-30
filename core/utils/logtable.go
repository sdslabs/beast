package utils

import (
	"os"

	"github.com/olekukonko/tablewriter"
)

type TableConfigs struct {
	TableBorders tablewriter.Border
	Separator    string
}

// LogTable logs table of the challenges in present the database
func LogTable(configs *TableConfigs, headers []string, logData [][]string) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(headers)
	table.SetBorders(configs.TableBorders)
	table.SetCenterSeparator(configs.Separator)
	table.AppendBulk(logData)
	table.Render()
}

func CreateTableConfigs(borders tablewriter.Border, separator string) *TableConfigs {
	tConfigs := &TableConfigs{
		TableBorders: borders,
		Separator:    separator,
	}
	return tConfigs
}
