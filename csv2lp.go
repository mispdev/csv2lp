package main

import (
	"fmt"
	"io"
	"os"

	"github.com/influxdata/influxdb/v2/pkg/csv2lp"
)

/**
 * This is just a small wrapper to call the CsvToLineProtocol function in the
 * csv2lp library provided in the InfluxDB sources. We just hand over the CSV
 * file and dump the line protocol output of the converter.
 */
func main() {

	args := os.Args[1:]

	if len(args) < 1 {
		fmt.Println("csv2lp")
		fmt.Println("------")
		fmt.Println("\nConvert annotated CSV as exported via Flux to InfluxDB line protocol.")
		fmt.Println("\nUsing csv2lp[1] library of InfluxDB.")
		fmt.Println("[1] https://github.com/influxdata/influxdb/tree/master/pkg/csv2lp")
		fmt.Println("\nSyntax: csv2lp <csv-file-name>")
		os.Exit(0)
	}

	filename := args[0]

	file, err := os.Open(filename)
	if err != nil {
		fmt.Println("Something went wrong opening the file:", err)
		os.Exit(1)
	} else {
		defer file.Close()
		reader := csv2lp.CsvToLineProtocol(file)
		buffer := make([]byte, 1024)
		for {
			n, e := reader.Read(buffer)
			if e != nil {
				if e == io.EOF {
					break
				} else {
					fmt.Println("Error:", e)
					break
				}
			}
			fmt.Print(string(buffer[:n]))
		}
	}

}
