package csv

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func Example_encoderEncodeStructs() {
	type myStruct struct {
		Field1 string `csv:"My Field 1"`
		Field2 string
	}
	var buf bytes.Buffer

	ss := []myStruct{
		{
			Field1: "Hello",
			Field2: "World",
		},
	}

	encoder := NewEncoder("|", "\n", &buf)
	encoder.WriteHeadersFor(ss[0])

	for _, rec := range ss {

		/* Validate the record
		if err := Validate(rec); err != nil {
			// handel error
			continue
		}
		*/
		if err := encoder.Encode(rec); err != nil {
			// handel error
			continue
		}
	}

	fmt.Print(buf.String())

}

func TestEncoder_Encode(t *testing.T) {
	var buf bytes.Buffer
	type myStruct struct {
		Field1 string `csv:"My Field 1"`
		Field2 string
	}
	simpleByValue := myStruct{
		Field1: "Hello",
		Field2: "World",
	}
	encoder := NewEncoder(",", "\n", &buf)
	if err := encoder.Encode(simpleByValue); err != nil {
		t.Fatalf("There was an error writing to the bytes buffer")
	}

	if buf.String() != "Hello,World\n" {
		t.Errorf("Encoder.Encode incorrectly encoded a struct to a deliniated format")
	}

}

func TestDecoder_Decode(t *testing.T) {

	type expectStruct struct {
		Field1 string
		Field2 string
	}

	tests := []struct {
		name   string
		input  string
		expect []expectStruct
	}{
		{
			name:  "One Simple Record",
			input: "Hello,World",
			expect: []expectStruct{
				{"Hello", "World"},
			},
		},
		{
			name:  "Two Simple Record",
			input: "Hello,World\nWelcome,Mars",
			expect: []expectStruct{
				{"Hello", "World"},
				{"Welcome", "Mars"},
			},
		},
		{
			name:  "Two Quoted Record",
			input: "\"Hel,lo\",World\nWelcome,Mars",
			expect: []expectStruct{
				{"Hel,lo", "World"},
				{"Welcome", "Mars"},
			},
		},
	}

	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {
			/*defer func() {
				if err := recover(); err != nil {
					t.Fatal(err)
				}
			}()*/
			reader := strings.NewReader(test.input)

			decoder := NewDecoder(",", "\n", reader)

			var records []expectStruct

			for decoder.Scan() {
				var decoded expectStruct
				if err := decoder.Decode(&decoded); err != nil {
					t.Fatalf("%v\n\t(%+v)\n\t(%s)",err, decoded, decoder.Text())
				}

				records = append(records, decoded)
				t.Log(decoded)

			}

			if ok := reflect.DeepEqual(records, test.expect); !ok {
				t.Errorf("Decode Failed: %s\n", test.name)
			}

		})

	}

}

func TestColsToRow(t *testing.T) {
	tests := []struct {
		cols []string
		row  string
	}{
		{
			cols: []string{"Hello World", "Column 2"},
			row:  `Hello World,Column 2`,
		},
		{
			cols: []string{"Hello, World", "Column 2"},
			row:  `"Hello, World",Column 2`,
		},
		{
			cols: []string{"Hello World", "Column 2", ""},
			row:  `Hello World,Column 2,`,
		},
		{
			cols: []string{"Hello World", "Column 2", ","},
			row:  `Hello World,Column 2,","`,
		},
		{
			cols: []string{},
			row:  ``,
		},
	}

	for _, test := range tests {
		t.Run(test.row, func(t *testing.T) {
			defer func() {
				if err := recover(); err != nil {
					t.Fatal(err)
				}
			}()
			row := colsToRow(test.cols, ",")
			if row != test.row {
				t.Errorf("Expected row: (%s), Got (%s)\n", test.row, row)
			}
		})
	}
}

func TestRowToCols(t *testing.T) {
	tests := []struct {
		row  string
		cols []string
	}{
		{
			row:  "Hello World,Column 2",
			cols: []string{"Hello World", "Column 2"},
		},
		{
			row:  `"Hello World",Column 2`,
			cols: []string{"Hello World", "Column 2"},
		},
		{
			row:  `"Hello World","Column 2"`,
			cols: []string{"Hello World", "Column 2"},
		},
		{
			row:  `"Hello World","Column 2",`,
			cols: []string{"Hello World", "Column 2", ""},
		},
		{
			row:  `,"Column 2",`,
			cols: []string{"", "Column 2", ""},
		},
		{
			row:  `Hello world,"Column 2",`,
			cols: []string{"Hello world", "Column 2", ""},
		},
		{
			row:  `Hello world,"Column 2",""`,
			cols: []string{"Hello world", "Column 2", ""},
		},
		{
			row: `Hello world,"Column
2",""`,
			cols: []string{"Hello world", "Column\n2", ""},
		},
	}

	for _, test := range tests {
		t.Run(test.row, func(t *testing.T) {
			defer func() {
				if err := recover(); err != nil {
					t.Fatal(err)
				}
			}()
			cols := rowToCols(test.row, ",")
			if ok := reflect.DeepEqual(cols, test.cols); !ok {
				t.Errorf("rowToCols Failed: (%s) -- [%s]\n", strings.Join(cols, "::"), strings.Join(test.cols, "::"))
			}
		})
	}
}

func TestIndexOfQuotedEndOfLine(t *testing.T) {
	tests := []struct {
		input string
		firstToken string
		expect int
	} {
		{
			input: "",
			firstToken:"",
			expect: -1,
		},
		{
			input: "\n",
			firstToken:"",
			expect: 0,
		},
		{
			input: "01234\n",
			firstToken:"01234",
			expect: 5,
		},
		{
			input: "\"\"234\n",
			firstToken: "\"\"234",
			expect: 5,
		},
		{
			input: "\"234\n\"",
			firstToken: "\"234\n\"",
			expect: -1,
		},
		{
			input: "\"234\n\"\n",
			firstToken: "\"234\n\"",
			expect: 6,
		},
		{
			input: "\"234\n\"\n12345",
			firstToken: "\"234\n\"",
			expect: 6,
		},
		{
			input: "\"Hel,lo\",World\nWelcome,Mars",
			firstToken: "\"Hel,lo\",World",
			expect: 14,
		},
	}

	for i, test := range tests{
		t.Run(test.input, func(t *testing.T) {
			if idx := indexOfQuotedEndOfLine([]byte(test.input)); idx != test.expect {
				t.Errorf("%d; expected %d. got %d", i, test.expect, idx)
			}

			_, token, err := ScanQuotedLine([]byte(test.input), true)
			if err != nil {
				t.Error(err)
			}
			if string(token) != test.firstToken {
				t.Errorf("got %s. expected %s", token, test.firstToken)
			}

		})
	}


}

/*
func TestDecode(t *testing.T){

	type myStruct struct {
		Field1 string `csv:"My Field 1"`
		Field2 string
	}

	var buf bytes.Buffer

	decoder := Decoder {
		r: &buf,
		nl: "\n",
		delimeter:"|",
		skipRows: 1,	// skips the header rows
	}

	var cursor myStruct
	for decoder.Scan() {
		if err := decoder.Decode(&cursor); err != nil {
			//handel error
			continue
		}

		// do something with cursor
		cursor...
	}

}

func TestColumnHeaders(t *testing.T) {
	type myStruct struct {
		Field1 string `csv:"My Field 1"`
		Field2 string
	}

	headersByValue := myStruct{
		Field1: "Data1",
		Field2: "Data2",
	}

	headers := ColumnHeaders(headersByValue)
	if len(headers) != 2 {
		t.Fatalf("Failed to get all headers")
	}

	if headers[0] != "My Field 1" {
		t.Errorf("Expected col1 to be 'My Field 1'. Got %s", headers[0])
	}

	if headers[1] != "Field2" {
		t.Errorf("Expected col1 to be 'Field2'. Got %s", headers[1])
	}

}

*/

/*
func TestEncoder_Encode(t *testing.T) {
	var buf bytes.Buffer

	type myStruct struct {
		Field1 string
		Field2 string
	}

	encoder := NewEncoder(&buf)

	simpleByValue := myStruct{
		Field1: "Hello",
		Field2: "World",
	}
	if err := encoder.Encode(simpleByValue); err != nil {
		t.Fatalf("There was an error writing to the bytes buffer")
	}

	if buf.String() != "Hello,World\n" {
		t.Errorf("Encoder.Encode incorrectly encoded a struct to a deliniated format")
	}

	buf.Reset()

	simpleByReference := myStruct{
		Field1: "Hello",
		Field2: "World",
	}

	if err := encoder.Encode(&simpleByReference); err != nil {
		t.Fatalf("There was an error writing to the bytes buffer")
	}

	if buf.String() != "Hello,World\n" {
		t.Errorf("Encoder.Encode incorrectly encoded a struct to a deliniated format")
	}
}
*/
