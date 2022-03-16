// Copyright 2015 Joel Scoble. All rights reserved.
// Use of thsi source code is governed by the MIT license that can
// be found in the LICENSE file.

// Some aspects of the code are written by The Go Authors.
// Original comment:
// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package struct2csv

import (
	"encoding/csv"
	"io"
)

// A Writer writes structs to a CSV encoded file.  This wraps both `csv.Writer`
// and this package's `Encoder`.
type Writer struct {
	e Encoder
	w *csv.Writer
	b int64
	r int
}

// NewWriter returns a new Writer that write to w.
func NewWriter(w io.Writer) *Writer {
	enc := New()
	return &Writer{e: *enc, w: csv.NewWriter(w)}
}

// WriteColNames writes out the column names of the CSV field.
func (w *Writer) WriteColNames(st interface{}) error {
	cols, err := w.e.GetColNames(st)
	if err != nil {
		return err
	}
	w.r++
	return w.w.Write(cols)
}

// WriteStruct takes a struct, marshals it to CSV and writhes the CSV
// record to the writer
func (w *Writer) WriteStruct(st interface{}) error {
	row, err := w.e.GetRow(st)
	if err != nil {
		return err
	}
	w.r++
	return w.w.Write(row)
}

// WriteStructs takes a slice of structs and writes them as CSV records.  This
// includes writing out the column names as the first row.  When done, Flush
// is called.
func (w *Writer) WriteStructs(st interface{}) error {
	rows, err := w.e.Marshal(st)
	if err != nil {
		return err
	}
	w.r = len(rows)
	w.w.WriteAll(rows)
	w.w.Flush()
	return w.w.Error()
}

// Write takes a slice of strings and writes them as a single CSV record.
func (w *Writer) Write(row []string) error {
	err := w.w.Write(row)
	if err != nil {
		return err
	}
	w.r++
	return nil
}

// WriteAll writes multiple CSV records, a two-d slice of strings,
// `[][]string` to w using Write and then calls Flush.
func (w *Writer) WriteAll(data [][]string) error {
	err := w.w.WriteAll(data)
	if err != nil {
		return err
	}
	w.r = len(data)
	return nil
}

// Flush writes any buffered data to the underlying io.Writer.
func (w *Writer) Flush() {
	w.w.Flush()
}

// Error reports an error that has occurred during a previous Write or Flush
func (w *Writer) Error() error {
	return w.w.Error()
}

// Rows returns the number of CSV rows created. This includes the header row.
func (w *Writer) Rows() int {
	return w.r
}

// Expose public CSV fields

// Comma is the field delimiter, set to '.'
func (w *Writer) Comma() rune {
	return w.w.Comma
}

// SetComma takes the passed rune and uses it to set the field
// delimiter for CSV fields.
func (w *Writer) SetComma(r rune) {
	w.w.Comma = r
}

// UseCRLF exposes the csv writer's UseCRLF field.
func (w *Writer) UseCRLF() bool {
	return w.w.UseCRLF
}

// SetUseCRLF set's the csv'writer's UseCRLF field
func (w *Writer) SetUseCRLF(b bool) {
	w.w.UseCRLF = b
}

// Expose Encoder methods

// SetTag set's the tag value to match on for a struct's field tags.
func (w *Writer) SetTag(s string) {
	w.e.SetTag(s)
}

// SetUseTags set's whether or not field tag values should be checked.
// If field tags are not being checked, the field name will be used for
// the column name.
func (w *Writer) SetUseTags(b bool) {
	w.e.SetUseTags(b)
}

// SetSeparators sets the begin and end separator values for lists, which
// default to `(` and `)`.  Empty strings result in no separators being used.
func (w *Writer) SetSeparators(beg, end string) {
	w.e.SetSeparators(beg, end)
}

// SetBase set's the base for _uint_ values; mainly used for
// `strings.FormatUint()`. By default, this is set to 10, for base 10
// numbering.  Any base value < 2 will be set to 2, binary.
func (w *Writer) SetBase(i int) {
	w.e.SetBase(i)
}

// ColNames returns a copy of the encoder's cached column names
func (w *Writer) ColNames() []string {
	return w.e.ColNames()
}
