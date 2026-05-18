package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	dsn := "ichsan:12345@tcp(151.243.222.93:30036)/archeris?parseTime=true"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("failed to open database connection: %v", err)
	}
	defer db.Close()

	// create file
	f, err := os.Create("scratch/db_info.txt")
	if err != nil {
		log.Fatalf("failed to create output file: %v", err)
	}
	defer f.Close()

	// ping the db
	err = db.Ping()
	if err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}
	fmt.Fprintln(f, "successfully connected to the archeris database")

	// 1. show tables
	rows, err := db.Query("SHOW TABLES")
	if err != nil {
		log.Fatalf("failed to query tables: %v", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			log.Fatalf("failed to scan table name: %v", err)
		}
		tables = append(tables, tableName)
	}

	fmt.Fprintf(f, "\nfound %d tables in the database:\n", len(tables))
	for _, table := range tables {
		// get row count for each table
		var count int
		err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM `%s`", table)).Scan(&count)
		if err != nil {
			fmt.Fprintf(f, "- %s (error counting rows: %v)\n", table, err)
			continue
		}
		fmt.Fprintf(f, "- %s: %d rows\n", table, count)
	}

	// 2. show schema for all tables
	fmt.Fprintln(f, "\n--- schemas for all tables ---")
	for _, table := range tables {
		fmt.Fprintf(f, "\ncolumns for table: %s\n", table)
		cRows, err := db.Query(fmt.Sprintf("DESCRIBE `%s`", table))
		if err != nil {
			fmt.Fprintf(f, "error describing table %s: %v\n", table, err)
			continue
		}
		defer cRows.Close()

		fmt.Fprintf(f, "%-25s | %-15s | %-5s | %-3s | %-10s | %-15s\n", "field", "type", "null", "key", "default", "extra")
		fmt.Fprintln(f, "--------------------------------------------------------------------------------------------------------")
		for cRows.Next() {
			var field, t, null, key sql.NullString
			var def, extra sql.NullString
			if err := cRows.Scan(&field, &t, &null, &key, &def, &extra); err != nil {
				fmt.Fprintf(f, "error scanning column: %v\n", err)
				break
			}
			fmt.Fprintf(f, "%-25s | %-15s | %-5s | %-3s | %-10s | %-15s\n",
				field.String, t.String, null.String, key.String, def.String, extra.String)
		}
	}
	fmt.Println("done writing to scratch/db_info.txt")
}
