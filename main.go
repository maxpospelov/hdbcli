package main

import (
	"log"
	"os/user"
	"path/filepath"

	"fmt"

	"errors"

	"os"

	"strings"

	"encoding/csv"

	"github.com/olekukonko/tablewriter"
	"github.com/xandout/gorpl"
	"github.com/xandout/hdbcli/config"
	"github.com/xandout/hdbcli/db"
)

var mode string = "table"

func tablePrinter(simpleRows *db.SimpleRows) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(simpleRows.Columns)
	table.AppendBulk(simpleRows.Rows)
	table.SetAlignment(tablewriter.ALIGN_RIGHT) // Set Alignment
	table.Render()

}

func csvPrinter(simpleRows *db.SimpleRows, printHeader bool) {
	if printHeader {
		w := csv.NewWriter(os.Stdout)

		header := [][]string{simpleRows.Columns}
		for _, record := range header {
			if err := w.Write(record); err != nil {
				log.Fatalln("error writing record to csv:", err)
			}
		}

		for _, record := range simpleRows.Rows {
			if err := w.Write(record); err != nil {
				log.Fatalln("error writing record to csv:", err)
			}
		}

		// Write any buffered data to the underlying writer (standard output).
		w.Flush()

		if err := w.Error(); err != nil {
			log.Fatal(err)
		}
	} else {
		w := csv.NewWriter(os.Stdout)

		for _, record := range simpleRows.Rows {
			if err := w.Write(record); err != nil {
				log.Fatalln("error writing record to csv:", err)
			}
		}

		// Write any buffered data to the underlying writer (standard output).
		w.Flush()

		if err := w.Error(); err != nil {
			log.Fatal(err)
		}
	}
}

func print(rows *db.SimpleRows) {
	if rows.Length == 0 {
		fmt.Println("No rows returned")
		return
	}
	switch mode {
	case "csv":
		csvPrinter(rows, true)
	case "table":
		tablePrinter(rows)
	}
}

func main() {
	u, userErr := user.Current()
	if userErr != nil {
		log.Fatal(userErr)
	}
	conf, err := config.LoadConfiguration(filepath.Join(u.HomeDir, ".hdbcli_config.json"))

	if err != nil {
		log.Fatal(err)
	}
	d, err := db.New(*conf)

	if err != nil {
		log.Fatal(err)
	}

	g := gorpl.New("> ", ";")
	g.AddAction("/describe", func(args ...interface{}) (interface{}, error) {
		fmtString := "SELECT COLUMN_NAME,DATA_TYPE_NAME,LENGTH,IS_NULLABLE, SCHEMA_NAME FROM TABLE_COLUMNS WHERE TABLE_NAME = '%s';"
		if len(args) != 1 {
			return nil, errors.New("describe function requires a table name to be supplied")
		}
		finalQ := fmt.Sprintf(fmtString, strings.ToUpper(args[0].(string)))
		fmt.Println(finalQ)
		g.RL.SaveHistory(finalQ)
		res, err := d.Run(finalQ)
		if err != nil {
			log.Println(err)
			return "", err
		}
		print(&res.SRows)
		return "", nil
	})
	g.AddAction("/exit", func(args ...interface{}) (interface{}, error) {
		fmt.Println("Bye!")
		os.Exit(0)
		return nil, nil
	})
	g.AddAction("/mode", func(args ...interface{}) (interface{}, error) {
		modes := map[string]bool{
			"csv":   true,
			"table": true,
		}
		if len(args) != 1 {
			return nil, errors.New("mode function requires a valid output mode")
		}
		attemptedMode := args[0]
		if modes[attemptedMode.(string)] {
			mode = attemptedMode.(string)
		} else {
			fmt.Println("Valid modes are:")
			for k := range modes {
				fmt.Printf("\t%s\n", k)
			}
		}
		return "", nil

	})
	g.Default = gorpl.Action{
		Action: func(args ...interface{}) (interface{}, error) {
			res, err := d.Run(args[0].(string))
			if err != nil {
				log.Println(err)
				return "", err
			}
			if res.Type == "query" {
				print(&res.SRows)
			} else {
				fmt.Printf("Affected Rows %v\n", res.RowsAffected)
				fmt.Printf("Last Insert ID: %v\n", res.LastInsertId)
			}

			return "", nil
		},
	}
	g.Start()

}
