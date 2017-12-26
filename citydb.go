package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
)

type CityEntry struct {
	GeoID   int
	Country string
	Name    string
}

type ByGeoId []CityEntry

func (b ByGeoId) Len() int {
	return len(b)
}
func (b ByGeoId) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}
func (b ByGeoId) Less(i, j int) bool {
	return b[i].GeoID < b[j].GeoID
}

func (e CityEntry) String() string {
	return fmt.Sprintf("%v: country=%v, name=%v", e.GeoID, e.Country, e.Name)
}

type CityDatabase struct {
	Source  string
	Entries []CityEntry
}

func NewCityDatabase(csvFilename string) (*CityDatabase, error) {
	f, err := os.Open(csvFilename)
	if err != nil {
		return nil, err
	}
	reader := csv.NewReader(f)

	db := CityDatabase{}
	db.Entries = make([]CityEntry, 0, 103546)
	reader.Read() // ignore the header line
	lineno := 1
	ignored := 0
	for {
		lineno++

		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		var entry CityEntry
		id, err := strconv.ParseInt(record[0], 10, 32)
		if err != nil {
			ignored++
			continue
		}

		entry.GeoID = int(id)
		entry.Country = record[4]
		entry.Name = record[10]

		db.Entries = append(db.Entries, entry)
	}
	log.Printf("parsed %v lines, %v lines ignored", lineno, ignored)

	sort.Sort(ByGeoId(db.Entries))
	log.Printf("sort finished")

	return &db, nil
}

func (b *CityDatabase) Search(id int) (CityEntry, error) {
	idx := sort.Search(len(b.Entries), func(i int) bool {
		return id <= b.Entries[i].GeoID
	})
	if idx == len(b.Entries) {
		return CityEntry{}, fmt.Errorf("no entry matched to %v", id)
	}
	return b.Entries[idx], nil
}
