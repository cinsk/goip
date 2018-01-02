package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
)

var ProgramName string

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
var formatter Formatter
var formatterName string
var fieldOrder string
var fieldSeparator string
var tcpAddress string
var numGroups int
var numGroupIteration int

func init() {
	ProgramName = path.Base(os.Args[0])

	flag.StringVar(&dbURL, "u", GEOLITE_ARCHIVE_URL, "url of MaxMind geolocation database (zip)")
	flag.StringVar(&dbDirectory, "d", "", "directory of GeoDB")
	flag.StringVar(&cityDBName, "c", GEOLITE_CITY_CSV_FILE, "city db filename")
	flag.StringVar(&blockDBName, "b", GEOLITE_BLOCK_CSV_FILE, "block db filename")
	flag.BoolVar(&noCleanUp, "n", false, "do not remove the downloaded files.")
	flag.BoolVar(&verboseMode, "v", false, "quiet mode")
	flag.BoolVar(&includeUnknown, "U", false, "do not remove unknown")
	flag.IntVar(&limitCount, "l", 1000, "print only top n elements")

	flag.StringVar(&inputFilename, "i", "", "do not remove the downloaded files.")

	flag.StringVar(&formatterName, "t", "csv", "formatter type: csv or text")
	flag.StringVar(&fieldSeparator, "f", "\t", "field separator for text formatter")
	flag.StringVar(&fieldOrder, "o", "name,pop,lat,lon,group", "field order of name, pop, lat, lon, and group")

	flag.StringVar(&tcpAddress, "T", "", "enable server mode, tcp address:port for listening socket")
	flag.IntVar(&numGroups, "g", 5, "number of groups for clustering the output")
	flag.IntVar(&numGroupIteration, "G", 20, "number of iteration for grouping/clustering")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTION...]\n", ProgramName)
		fmt.Fprintf(os.Stderr, "Print Geolocation of given IP addresses\n")
		fmt.Fprintf(os.Stderr, "\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n")
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
	log.Printf("formatter: %v", formatterName)
	log.Printf("fieldOrder: %v", fieldOrder)
	log.Printf("nGroup: %v", numGroups)
	log.Printf("nGroupIteration: %v", numGroupIteration)
	log.Printf("limitCount: %v", limitCount)

	formatter, err := NewFormatter(formatterName, fieldOrder, fieldSeparator)
	if err != nil {
		Err(1, err, "cannot create a formatter")
	}

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

	CityDB, err := NewCityDatabase(path.Join(dbDirectory, cityDBName))
	if err != nil {
		Err(1, err, "cannot load city database")
	}
	BlockDB, err = NewBlockDatabase(path.Join(dbDirectory, blockDBName), CityDB)
	if err != nil {
		Err(1, err, "cannot load block database")
	}

	server := NewServer()
	server.Start()
	if tcpAddress != "" {
		server.AddListener("tcp", tcpAddress)
		fmt.Fprintf(os.Stderr, "server ready\n")
	}
	defer server.Close()

	scanner := bufio.NewScanner(inputFile)

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt)
	signal.Notify(signalChannel, os.Kill)

	stdinDone := make(chan struct{})
	go func() {
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			// fmt.Fprintf(os.Stderr, "# line: %s\n", line)
			if line == "" {
				continue
			}

			server.Incoming <- LocationRequest{Address: line}
		}
		if err := scanner.Err(); err != nil {
			Err(1, err, "reading the file %v", inputFile.Name())
		}

		done := make(chan struct{})
		NewFormatter(formatterName, fieldOrder, fieldSeparator)
		server.Incoming <- StatisticRequest{
			Limit:             limitCount,
			Stream:            os.Stdout,
			Formatter:         formatter,
			Done:              done,
			Groups:            numGroups,
			MaxGroupIteration: numGroupIteration,
		}
		<-done
		close(stdinDone)
	}()

	select {
	case <-stdinDone:
		if !noCleanUp {
			downloader.Close()
		}
	case c := <-signalChannel:
		if !noCleanUp {
			downloader.Close()
		}
		fmt.Fprintf(os.Stderr, "received a signal: %v\n", c)

		if sig, ok := c.(syscall.Signal); ok {
			os.Exit(128 + int(sig))
		} else {
			os.Exit(128)
		}
	}
}
