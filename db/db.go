package db

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	_ "github.com/SAP/go-hdb/driver"
	"github.com/xandout/hdbcli/config"
)

type DBR struct {
	SRows        SimpleRows
	RowsAffected int64
	LastInsertId int64
	Type         string
}

type DB struct {
	connection *sql.DB
}

type SimpleRows struct {
	Columns []string
	Rows    [][]string
	Length  int
}

// convertRows takes *sql.Rows and converts it to a SimpleRows
func convertRows(rows *sql.Rows) (simpleRows *SimpleRows, err error) {
	simpleRows = new(SimpleRows)
	columns, err := rows.Columns()
	colLen := len(columns)
	if err != nil {
		return simpleRows, err
	}
	if colLen > 0 {
		simpleRows.Columns = columns
	} else {
		return simpleRows, errors.New("got 0 columns")
	}
	values := make([]interface{}, colLen)
	scanArgs := make([]interface{}, colLen)
	if err != nil {
		return simpleRows, err
	}
	for i := range values {
		scanArgs[i] = &values[i]
	}

	for rows.Next() {
		colVals := make([]string, colLen)
		err := rows.Scan(scanArgs...)
		if err != nil {
			log.Fatal(err)
		}
		for i, v := range values {
			switch t := v.(type) {
			case float64:
				colVals[i] = fmt.Sprintf("%f", t)
			case bool:
				colVals[i] = fmt.Sprintf("%t", t)
			case string:
				colVals[i] = fmt.Sprintf("%s", t)
			case int64:
				colVals[i] = fmt.Sprintf("%d", t)
			case nil:
				colVals[i] = "NULLL"
			case []uint8:
				colVals[i] = fmt.Sprintf("%s", []byte(t[:]))
			case time.Time:
				colVals[i] = fmt.Sprintf("%s", t)
			default:
				colVals[i] = fmt.Sprintf("%v", t)

			}
		}
		simpleRows.Rows = append(simpleRows.Rows, colVals)
		simpleRows.Length++
	}
	return simpleRows, nil

}

func (db *DB) identifyType(statement string) string {
	if strings.HasPrefix(strings.ToUpper(statement), "SELECT") {
		return "query"
	} else {
		return "exec"
	}
}

func (db *DB) exec(statement string) (map[string]interface{}, error) {
	var ret map[string]interface{}
	res, err := db.connection.Exec(statement)
	if err != nil {
		return nil, err
	}
	ra, raErr := res.RowsAffected()
	li, liErr := res.LastInsertId()

	if raErr != nil {
		return nil, raErr
	}
	if liErr != nil {
		return nil, liErr
	}
	ret["RowsAffected"] = ra
	ret["LastInsertId"] = li
	return ret, nil

}

func (db *DB) query(statement string) (*sql.Rows, error) {
	rows, err := db.connection.Query(statement)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (db *DB) Run(statement string) (DBR, error) {
	var dbr DBR
	t := db.identifyType(statement)
	switch t {
	case "query":
		dbr.Type = "query"
		r, err := db.query(statement)
		if err != nil {
			return DBR{}, err
		}
		sr, err := convertRows(r)
		if err != nil {
			return DBR{}, err
		}
		dbr.SRows = *sr
		return dbr, nil
	case "exec":
		dbr.Type = "exec"
		res, err := db.exec(statement)
		if err != nil {
			return DBR{}, nil
		}
		dbr.RowsAffected = res["RowsAffected"].(int64)
		dbr.LastInsertId = res["LastInsertId"].(int64)
		return dbr, nil
	}
	return DBR{}, errors.New("should have never got here db.Run")
}

func getDbURL(configuration config.Configuration) string {

	dsn := &url.URL{
		Scheme: "hdb",
		User:   url.UserPassword(configuration.Username, configuration.Password),
		Host:   fmt.Sprintf("%s:%d", configuration.Hostname, configuration.Port),
	}
	return dsn.String()
}

func New(config config.Configuration) (DB, error) {
	var db DB
	conn, err := sql.Open("hdb", getDbURL(config))
	if err != nil {
		return DB{}, err
	}
	db.connection = conn
	if db.connection.Ping() != nil {
		log.Fatal(err)
		return DB{}, err
	}
	return db, nil
}
