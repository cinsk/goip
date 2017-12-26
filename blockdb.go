package main

import (
	"encoding/binary"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sort"
	"strconv"
)

type BlockEntry struct {
	IP4Range
	GeoID     int
	Latitude  float32
	Longitude float32
	City      CityEntry
}

type ByBegin []BlockEntry

func (b ByBegin) Len() int {
	return len(b)
}
func (b ByBegin) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}
func (b ByBegin) Less(i, j int) bool {
	return b[i].Begin < b[j].Begin
}

func (e BlockEntry) String() string {
	return fmt.Sprintf("%s-%s: id=%v, location=(%f, %f), city=(%v)", int2ip(e.Begin), int2ip(e.End), e.GeoID, e.Longitude, e.Latitude, e.City)
	// return fmt.Sprintf("%10d-%10d: id=%v, location=(%f, %f)",
	// 	e.Begin, e.End, e.GeoID, e.Longitude, e.Latitude)
}

type BlockDatabase struct {
	Source  string
	CityDB  *CityDatabase
	Entries []BlockEntry
}

func NewBlockDatabase(csvFilename string, cityDB *CityDatabase) (*BlockDatabase, error) {
	f, err := os.Open(csvFilename)
	if err != nil {
		return nil, err
	}
	reader := csv.NewReader(f)

	db := BlockDatabase{}
	db.Entries = make([]BlockEntry, 0, 2711472)
	db.CityDB = cityDB

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

		var entry BlockEntry
		entry.IP4Range, err = NewIP4Range(record[0])
		if err != nil {
			// log.Printf("%d: cannot parse %v as CIDR, ignored: %v", lineno, record[0], err)
			ignored++
			continue
		}

		geoid, err := strconv.ParseUint(record[1], 10, 32)
		if err != nil {
			// log.Printf("%d: cannot parse %v as uint32, ignored: %v", lineno, record[1], err)
			ignored++
			continue
		}
		entry.GeoID = int(geoid)

		lat, err := strconv.ParseFloat(record[7], 32)
		if err != nil {
			// log.Printf("%d: cannot parse %v as float32, ignored: %v", lineno, record[7], err)
			ignored++
			continue
		}
		entry.Latitude = float32(lat)

		lng, err := strconv.ParseFloat(record[8], 32)
		if err != nil {
			// log.Printf("%d: cannot parse %v as float32, ignored: %v", lineno, record[8], err)
			ignored++
			continue
		}
		entry.Longitude = float32(lng)

		db.Entries = append(db.Entries, entry)
	}
	log.Printf("parsed %v lines, %v lines ignored", lineno, ignored)

	sort.Sort(ByBegin(db.Entries))
	log.Printf("sort finished")

	for i := 0; i < len(db.Entries); i++ {
		city, err := db.CityDB.Search(db.Entries[i].GeoID)
		if err != nil {
			log.Printf("no city entry for geoID %v", db.Entries[i].GeoID)
			continue
		}

		db.Entries[i].City = city
	}

	return &db, nil
}

func (b *BlockDatabase) Search(ip string) (BlockEntry, error) {
	t := net.ParseIP(ip)
	if t == nil {
		return BlockEntry{}, fmt.Errorf("cannot parse IP address in %v", ip)
	}
	target := binary.BigEndian.Uint32(t[len(t)-4:])

	idx := sort.Search(len(b.Entries), func(i int) bool {
		return target <= b.Entries[i].End
	})
	if idx == len(b.Entries) {
		return BlockEntry{}, fmt.Errorf("no entry matched to %v (%d)", ip, target)
	}
	return b.Entries[idx], nil
}
