package main

import (
	"io/ioutil"
	"strings"
)

func main() {
	path := "c:\\E\\ichsan\\startup\\archeryhub.id\\api\\models\\event.go"
	content, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}

	s := string(content)
	
	// Systematic replacement
	s = strings.ReplaceAll(s, "ID           string    `json:\"id\" db:\"id\"`", "UUID         string    `json:\"id\" db:\"uuid\"` ")
	s = strings.ReplaceAll(s, "ID                string     `json:\"id\" db:\"id\"`", "UUID              string     `json:\"id\" db:\"uuid\"` ")
	s = strings.ReplaceAll(s, "ID                  string    `json:\"id\" db:\"id\"` ", "UUID                string    `json:\"id\" db:\"uuid\"` ")
	
	err = ioutil.WriteFile(path, []byte(s), 0644)
	if err != nil {
		panic(err)
	}
}
