package utils

import (
	"bytes"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/mohae/struct2csv"
)

// StructToCSV returns buffer stream from a struct to be disposed 
// as a CSV file in the HTTP response
func StructToCSV(c *gin.Context, st interface{}, filename string) (bytes.Buffer, error) {
	buff := &bytes.Buffer{}
	w := struct2csv.NewWriter(buff)

	if err := w.WriteStructs(st); err != nil {
		return *buff, err
	}

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	return *buff, nil
}
