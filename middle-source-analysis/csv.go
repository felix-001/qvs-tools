package main

import (
	"encoding/csv"
	"log"
	"os"
)

func (s *Parser) locadCsv(filename string) (rows [][]string) {
	/*
		bytes, err := ioutil.ReadFile(filename)
		if err != nil {
			log.Println("read fail", filename, err)
			return nil
		}
		lines := strings.Split(string(bytes), "\n")
		for _, line := range lines {
			columns := strings.Split(line, ",")
			rows = append(rows, columns)
		}
		return
	*/
	file, err := os.Open(filename)
	if err != nil {
		log.Println(err)
		return nil
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1 // Allow variable number of fields

	data, err := reader.ReadAll()
	if err != nil {
		log.Println(err)
		return nil
	}
	return data
}
