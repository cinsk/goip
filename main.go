package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"sort"
	"strings"
)

const GEOLITE_ARCHIVE_URL = "http://geolite.maxmind.com/download/geoip/database/GeoLite2-City-CSV.zip"
const GEOLITE_BLOCK_CSV_FILE = "GeoLite2-City-Blocks-IPv4.csv"
const GEOLITE_CITY_CSV_FILE = "GeoLite2-City-Locations-en.csv"

var CityDB *CityDatabase
var BlockDB *BlockDatabase

var dbDirectory string
var dbURL string
var cityDBName string
var blockDBName string
var noCleanUp bool
var inputFile *os.File
var inputFilename string
var verboseMode bool
var limitCount int
var includeUnknown bool

type PopulationEntry struct {
	Name      string
	Latitude  float32
	Longitude float32
	Count     int
}

type ByPopulation []PopulationEntry

func (p ByPopulation) Len() int           { return len(p) }
func (p ByPopulation) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p ByPopulation) Less(i, j int) bool { return p[i].Count > p[j].Count }
func init() {
	flag.StringVar(&dbURL, "u", GEOLITE_ARCHIVE_URL, "directory of GeoDB")
	flag.StringVar(&dbDirectory, "d", "", "directory of GeoDB")
	flag.StringVar(&cityDBName, "c", GEOLITE_CITY_CSV_FILE, "city db filename")
	flag.StringVar(&blockDBName, "b", GEOLITE_BLOCK_CSV_FILE, "block db filename")
	flag.BoolVar(&noCleanUp, "n", false, "do not remove the downloaded files.")
	flag.BoolVar(&verboseMode, "v", false, "quiet mode")
	flag.BoolVar(&includeUnknown, "U", false, "do not remove unknown")
	flag.IntVar(&limitCount, "l", -1, "print only top n elements")

	flag.StringVar(&inputFilename, "i", "", "do not remove the downloaded files.")
}

func Err(exitStatus int, err error, format string, args ...interface{}) {
	b := bytes.Buffer{}

	b.WriteString("error: ")
	b.WriteString(fmt.Sprintf(format, args...))

	if err != nil {
		b.WriteString(fmt.Sprintf(": %v", err))
	}

	os.Stdout.Sync()
	fmt.Fprintf(os.Stderr, "%s\n", b.String())
	os.Stderr.Sync()

	if exitStatus != 0 {
		os.Exit(exitStatus)
	}
}

func main() {
	if os.Getenv("DEBUG") == "" {
		log.SetOutput(ioutil.Discard)
	}

	flag.Parse()
	log.Printf("dbDirectory: %v", dbDirectory)
	log.Printf("cityDBName: %v", cityDBName)
	log.Printf("blockDBName: %v", blockDBName)
	log.Printf("os.Args: %v", os.Args)
	log.Printf("flag.Args: %v", flag.Args())

	downloader := Downloader{}
	if dbDirectory == "" {
		err := downloader.Fetch(dbURL)
		if err != nil {
			Err(1, err, "cannott fetch url, %s", dbURL)
		}
		err = downloader.Unpack()
		if err != nil {
			Err(1, err, "cannot unpack the archive")
		}

		dbDirectory = downloader.Base

		if noCleanUp {
			defer downloader.Close()
		}
	}

	if inputFilename == "" {
		inputFile = os.Stdin
	} else {
		f, err := os.Open(inputFilename)
		if err != nil {
			Err(1, err, "cannot open the input file %s", inputFilename)
		}
		defer f.Close()
		inputFile = f
	}

	var err error
	CityDB, err = NewCityDatabase(path.Join(dbDirectory, cityDBName))
	if err != nil {
		Err(1, err, "cannot load city database")
	}
	BlockDB, err = NewBlockDatabase(path.Join(dbDirectory, blockDBName), CityDB)
	if err != nil {
		Err(1, err, "cannot load block database")
	}

	scanner := bufio.NewScanner(inputFile)
	population := map[string]PopulationEntry{}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// fmt.Fprintf(os.Stderr, "# line: %s\n", line)

		if line == "" {
			continue
		}

		entry, err := BlockDB.Search(line)
		if err != nil && verboseMode {
			Err(0, err, "no entry for %s, ignored", line)
			continue
		}

		co, ci := entry.City.Country, entry.City.Name
		if !includeUnknown && (co == "" || ci == "") {
			continue
		}

		if co == "" {
			co = "UNKNOWN"
		}
		if ci == "" {
			ci = "UNKNOWN"
		}

		key := fmt.Sprintf("%v: %v", co, ci)
		if ent, ok := population[key]; ok {
			ent.Count += 1
			population[key] = ent
		} else {
			population[key] = PopulationEntry{Name: key, Count: 1, Latitude: entry.Latitude, Longitude: entry.Longitude}
		}

	}
	if err := scanner.Err(); err != nil {
		Err(1, err, "reading the file %v", inputFile.Name())
	}

	fmt.Printf("name,pop,lat,lon\n")
	// for k, v := range population {
	// 	fmt.Printf("'%v',%v,%v,%v\n", k, v.Count, v.Longitude, v.Latitude)
	// }

	entries := make([]PopulationEntry, 0, len(population))
	for _, v := range population {
		entries = append(entries, v)
	}
	sort.Sort(ByPopulation(entries))

	if limitCount < 0 {
		limitCount = len(entries)
	}
	for i := 0; i < limitCount; i++ {
		fmt.Printf("\"%v\",%v,%v,%v\n", entries[i].Name, entries[i].Count, entries[i].Latitude, entries[i].Longitude)
	}
}
