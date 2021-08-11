package csv

// TODO(tbshill): Move this outside of gocdrm into a public gopackage
// TODO(tbshill): Encoding multiline elements

import (
	"bufio"
	"fmt"
	"io"
	"reflect"
	"strings"
	"errors"
)

const CSVTagName = "csv"

var (
	ErrInvalidSchema = errors.New("row did not have the same number of fields as the struct")
)

// Decoder is a structure that will read a deliniated string into a structure
type Decoder struct {
	del string
	nl  string
	s   *bufio.Scanner
	row string
}

func ColsToRow(cols []string, del string) string {
	return colsToRow(cols, del)
}

func RowToCols(row, del string) []string {
	return rowToCols(row, del)
}

func wrapInQuotes(s string) string {
	return "\"" + s + "\""
}

func containsDelimeter(text, del string) bool {
	return strings.Contains(text, del)
}

func wrapInQuotesIfTextContainsDelimeter(text, del string) string {
	if containsDelimeter(text, del) {
		return wrapInQuotes(text)
	}

	return text
}

func colsToRow(cols []string, del string) string {

	if len(cols) == 0 {
		return ""
	}

	if len(cols) == 1 {
		return wrapInQuotesIfTextContainsDelimeter(cols[0], del)
	}

	var sb strings.Builder
	sb.WriteString(wrapInQuotesIfTextContainsDelimeter(cols[0], del))

	for _, col := range cols[1:] {
		sb.WriteString(del)
		sb.WriteString(wrapInQuotesIfTextContainsDelimeter(col, del))
	}

	return sb.String()

}

type rowToColState int

const (
	columnStartState rowToColState = iota
	quotedInnerState
	secondQuoteState
	innerColState
)
func rowToCols(row, del string) []string {

	var sb strings.Builder
	var cols []string

	doColumnStartState := func(next string) rowToColState {
		switch next {
		case del:
			cols = append(cols, sb.String())
			sb.Reset()
			return columnStartState
		case "\"":
			return quotedInnerState
		default:
			sb.WriteString(next)
			return innerColState
		}
	}

	doQuotedInnerState := func(next string) rowToColState {
		switch next {
		case "\"":
			return secondQuoteState
		default:
			sb.WriteString(next)
			return quotedInnerState
		}
	}

	doSecondQuoteState := func(next string) rowToColState {
		switch next {
		case del:
			cols = append(cols, sb.String())
			sb.Reset()
			return columnStartState
		default:
			panic("comma was expected at the end of a column, or support escaped quotes")
		}
	}

	doInnerColState := func(next string) rowToColState {
		switch next {
		case del:
			cols = append(cols, sb.String())
			sb.Reset()
			return columnStartState
		default:
			sb.WriteString(next)
			return innerColState
		}
	}

	state := columnStartState

	for _, v := range row {
		switch state {
		case columnStartState:
			state = doColumnStartState(string(v))
		case quotedInnerState:
			state = doQuotedInnerState(string(v))
		case secondQuoteState:
			state = doSecondQuoteState(string(v))
		case innerColState:
			state = doInnerColState(string(v))
		}
	}

	cols = append(cols, sb.String())
	sb.Reset()

	return cols
}

func ScanQuotedLine(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	if i := indexOfQuotedEndOfLine(data); i >= 0 {
		return i + 1, dropCR(data[0:i]), nil
	}

	if atEOF {
		return len(data), dropCR(data), nil
	}

	return 0, nil, nil
}

func indexOfQuotedEndOfLine(data []byte) int {
	char := 0
	quoted := 1

	state := char

	for i, r := range data {
		switch state {
		case char:
			switch r {
			case '\n':
				return i
			case '"':
				state = quoted
			}
		case quoted:
			switch r {
			case '"':
				state = char
			}
		}
	}

	return -1
}

func dropCR(data []byte) []byte {
	if len(data) > 0 && data[len(data)-1] == '\r' {
		return data[0: len(data)-1]
	}
	return data
}

// NewDecoder returns a new csv Decoder. 'del' is the delimeter, 'nl' is the
// newline character, and r is the source to read from
func NewDecoder(del, nl string, r io.Reader) *Decoder {
	decoder := &Decoder{
		del: del,
		nl:  nl,
		s:   bufio.NewScanner(r),
	}

	decoder.s.Split(ScanQuotedLine)
	return decoder
}

// Decode takes the string record that was loaded by Scan() and marshals it
// into a struct. The struct must be a reference, otherwise this function will
// panic
func (d *Decoder) Decode(obj interface{}) error {
	columns := rowToCols(d.row, d.del)

	v := reflect.ValueOf(obj).Elem()

	if v.NumField() != len(columns) {
		return ErrInvalidSchema
	}

	for i, column := range columns {
		v.Field(i).SetString(column)
	}

	return nil
}

// Scan reads a string record into a buffer to be decoded. It returns true if
// it found a record or false if it reached the end of the input stream
func (d *Decoder) Scan() bool {
	ok := d.s.Scan()
	if ok {
		d.row = d.s.Text()
	}

	return ok
}

func (d *Decoder) Text() string {
	return d.row
}

// Encoder is a utility that will encode a structure into a deliniated string.
// Like a csv
type Encoder struct {

	// del is the delimeter used to separate fields
	del string

	// nl is the newline character used to separate records.
	nl string

	// w is the destination to write the record
	w io.Writer
}

// NewEncoder creates a new csv encoder. del is the delimeter used, nl is the
// newline characters used, and w is the destination to write to
func NewEncoder(del, nl string, w io.Writer) Encoder {
	return Encoder{
		del: del,
		nl:  nl,
		w:   w,
	}
}

// WriteHeadersFor writes a record for the column headers. This should be the
// first record written
func (e Encoder) WriteHeadersFor(obj interface{}) error {
	columns := reflectColumnHeaders(obj)
	row := colsToRow(columns, e.del)
	_, err := fmt.Fprintf(e.w, "%s%s", row, e.nl)
	return err
}

// Encode writes a single record to the encoder. obj is reflected to obtain all
// of the values from the struct.  obj may be either passed by value or
// reference. obj may not be a chan, slice, or pimitive like int, string or
// float.
func (e Encoder) Encode(obj interface{}) error {
	v := reflect.ValueOf(obj)

	// If the obj is a reference
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	columns := make([]string, v.NumField())
	for i, _ := range columns {
		columns[i] = fmt.Sprintf("%v", v.Field(i).Interface())
	}

	row := colsToRow(columns, e.del)
	_, err := fmt.Fprintf(e.w, "%s%s", row, e.nl)
	return err
}

// reflectColumnHeaders returns the column headers for a specified struct. A
// column header is the name of the field or if the field has the "csv" tag,
// then it will use it's value
func reflectColumnHeaders(obj interface{}) []string {
	t := reflect.TypeOf(obj)

	// If the obj is a reference
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	headers := make([]string, t.NumField())

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if headerVal, ok := field.Tag.Lookup(CSVTagName); ok {
			headers[i] = headerVal
		} else {
			headers[i] = field.Name
		}
	}

	return headers
}
