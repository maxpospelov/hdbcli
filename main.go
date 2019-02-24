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
	"github.com/xandout/gorpl/action"
	"github.com/xandout/hdbcli/config"
	"github.com/xandout/hdbcli/db"
	"flag"
)

var (
	mode = "table"
	useConfigFile = flag.Bool("useConfig", false, "Read config from json file.")
	hostname = flag.String("h", "", "Hostname.")
	port = flag.Int("port", 39013, "Port number.")
	username = flag.String("u", "", "Username for connection.")
	password = flag.String("p", "", "Password for comnnection.")
	database = flag.String("d", "", "Database.")
	)

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

	d, err := connection(u)

	if err != nil {
		log.Fatal(err)
	}

	g := gorpl.New(";")

	exitAction := action.New("/exit", func(args ...interface{}) (interface{}, error) {
		fmt.Println("Bye!")
		os.Exit(0)
		return nil, nil
	})
	modeAction := action.New("/mode", func(args ...interface{}) (interface{}, error) {
		if len(args) == 0 {
			fmt.Printf("Current mode is %s\n", mode)
		}
		return "", nil
	})
	csvAction := action.New("csv", func(args ...interface{}) (interface{}, error) {
		mode = "csv"
		fmt.Printf("Mode set to %s\n", mode)

		return "", nil
	})
	tableAction := action.New("table", func(args ...interface{}) (interface{}, error) {
		mode = "table"
		fmt.Printf("Mode set to %s\n", mode)
		return "", nil
	})
	describeAction := action.New("/describe", func(args ...interface{}) (interface{}, error) {
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
	schemasAction := action.New("/schemas", func(args ...interface{}) (interface{}, error) {
		finalQ := "SELECT * FROM SCHEMAS;"

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

	//Register children Actions
	modeAction.AddChild(csvAction)
	modeAction.AddChild(tableAction)
	//Register parent Actions
	g.AddAction(*modeAction)
	g.AddAction(*exitAction)
	g.AddAction(*describeAction)
	g.AddAction(*schemasAction)
	g.Default = action.Action{
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

func connection(u *user.User) (*db.DB, error) {
	var err error
	var configuration *config.Configuration

	flag.Parse()

	if *useConfigFile {
		configuration, err = config.LoadConfiguration(filepath.Join(u.HomeDir, ".hdbcli_config.json"))
		} else {
		configuration = config.NewConfiguration(*hostname, *port, *username, *password, *database)
	}

	if err != nil {
		return nil, err
	}

	connection, err := db.New(*configuration)
	return &connection, err
}
