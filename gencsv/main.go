package main

import (
	"time"
	"fmt"
	"flag"
	"strings"
	"os"
	"io"
)


var template = `
package %s

// DO NOT EDIT - this file was generated with
// %s
// On %s

import (
	"github.com/tbshill/csv"
	"sync"
	"os"
	"time"
	%s
)

type %s struct {
	LoadedTimestamp time.Time
	Filename string
	Rownum int
	Errs []error
	Data %s
}

func New%s(filename string, row int, data %s) *%s {
	return  &%s {
			Filename: filename,
			Rownum: row,
			Data: data,
			Errs: []error{},
			LoadedTimestamp: time.Now(),
		}
}

func Load%s(filenames []string, parallelism int, dl, nl string) (chan *%s, chan error){
	recordChan, errChan := make(chan *%s), make(chan error)
	go func(){
		defer close(recordChan)
		defer close(errChan)

		concurrencyLimit := make(chan struct{}, parallelism)
		defer close(concurrencyLimit)

		var wg sync.WaitGroup
		wg.Add(len(filenames))

		for _, filename := range filenames {
			concurrencyLimit <- struct{}{}
			go func(wg *sync.WaitGroup, filename string) {

				defer func(){
					<-concurrencyLimit
				}()

				defer wg.Done()

				f, err := os.Open(filename)
				if err != nil {
					errChan <- err
					return
				}

				defer func(){
					if err := f.Close(); err != nil{
						errChan <- err
					}
				}()

				rowNum := 0
				decoder := csv.NewDecoder(dl, nl, f)
				for decoder.Scan() {
					rowNum++
					var data %s
					if err := decoder.Decode(&data); err != nil {
						errChan <- err
						return	// Parser is probably confused so we should exit
					}
					recordChan <- New%s(filename, rowNum, data)
				}

			}(&wg, filename)
		}

		wg.Wait()
	}()
	return recordChan, errChan
}
`

func generateCsvRecordName(dataType string) string {
	normalizedDatatypeName := dataType
	if idx := strings.LastIndexByte(dataType,'.'); idx >= 0 {
		normalizedDatatypeName = dataType[idx+1:]
	}
	return fmt.Sprintf("CsvRecord%s", normalizedDatatypeName)
}

func generateImports(imports []string) string {
	if len(imports) > 0 && imports[0] != ""{
		quoted := make([]string, len(imports))
		for i, v := range imports{
			quoted[i] = "\""+v+"\""
		}
		return strings.Join(quoted,"\n\t")
	}
	return ""
}

func generateTemplate(out io.Writer, genCommand, pkg, dataType string, imports []string) {
	csvRecordName := generateCsvRecordName(dataType)
	importStr := generateImports(imports)
	fmt.Fprintf(out, template,
		pkg,
		genCommand,
		time.Now().String(),
		importStr, // 1
		csvRecordName, // 2
		dataType, // 3
		csvRecordName, // 4
		dataType, // 5
		csvRecordName, // 6
		csvRecordName, // 6
		csvRecordName, // 6
		csvRecordName, // 7
		csvRecordName,
		dataType,
		csvRecordName) // 8
}

var (
	outfileName string
	pkg string
	dataType string
	importsRaw string
)

func main() {
	flag.StringVar(&outfileName, "out", "", "-out csvloader.go")
	flag.StringVar(&pkg, "pkg", "", "-pkg main")
	flag.StringVar(&dataType, "data", "", "-data schema.CareCloudLabExtractRecord")
	flag.StringVar(&importsRaw, "imports", "", "-imports \"github.com/tbshillcdr/gocdrm/schema,secondImport.com/...\"")
	flag.Parse()

	var out io.Writer
	if outfileName == "" {
		out = os.Stdout
	} else {
		var err error
		out, err = os.OpenFile(outfileName,os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0655)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create outfile with reason:%v", err)
		}
	}

	if pkg == "" {
		fmt.Fprintf(os.Stderr, "-pkg is requried\n")
		return
	}

	if dataType == "" {
		fmt.Fprintf(os.Stderr, "-data is requried\n")
		return
	}

	generateTemplate(out, strings.Join(os.Args, " "), pkg, dataType, strings.Split(importsRaw, ","))
}
