package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const READ_TIMEOUT_SECONDS = 5

type Request interface {
}

type LocationRequest struct {
	Address string
	Result  chan BlockEntry
}

type StatisticRequest struct {
	Limit             int
	Groups            int
	MaxGroupIteration int
	Stream            io.Writer
	Formatter         Formatter
	Done              chan struct{}
}

type ResetRequest struct{}

type Server struct {
	Groups int

	population map[string]PopulationEntry

	serverGroup sync.WaitGroup

	Incoming chan Request

	quitChannel   chan struct{}
	Listeners     []net.Listener
	listenerGroup sync.WaitGroup

	workerGroup sync.WaitGroup
}

func NewServer() *Server {
	return &Server{
		population:  make(map[string]PopulationEntry),
		Incoming:    make(chan Request),
		quitChannel: make(chan struct{}),
	}
}

func (s *Server) serveLocation(r LocationRequest) {
	entry, err := BlockDB.Search(r.Address)
	if err != nil {
		if verboseMode {
			Err(0, err, "no entry for %s, ignored", r.Address)
		}
		if r.Result != nil {
			r.Result <- BlockEntry{Error: err}
		}
		return
	}
	if r.Result != nil {
		r.Result <- entry
	}

	co, ci := entry.City.Country, entry.City.Name
	if !includeUnknown && (co == "" || ci == "") {
		return
	}
	if co == "" {
		co = "UNKNOWN"
	}
	if ci == "" {
		ci = "UNKNOWN"
	}
	key := fmt.Sprintf("%v: %v", co, ci)
	if ent, ok := s.population[key]; ok {
		ent.Count += 1
		s.population[key] = ent
	} else {
		s.population[key] = PopulationEntry{Name: key, Count: 1, Latitude: entry.Latitude, Longitude: entry.Longitude}
	}
}

type Centroid struct {
	Mean float64
}

type Centroids struct {
	Group map[int]Centroid
}

func Group(entries []PopulationEntry, ngroup int, maxIteration int) {
	centroids := NewCentroids(entries, ngroup)
	log.Printf("initial centroids: %v", centroids.Group)

	for i := 0; i < len(entries); i++ {
		id := centroids.Nearest(entries[i].Count)
		entries[i].Group = id
	}
	log.Printf("centroids: %v", centroids.Group)
	centroids.Reset(entries)
	log.Printf("centroids: %v", centroids.Group)

	for repeat := 0; repeat < maxIteration; repeat++ {
		log.Printf("-----: iteration=%v", repeat)
		total := 0
		for i := 0; i < len(entries); i++ {
			newId := centroids.Nearest(entries[i].Count)

			if newId != entries[i].Group {
				entries[i].Group = newId
				total++
			}
		}
		log.Printf("updated: %v entries", total)
		centroids.Reset(entries)
		log.Printf("centroids: %v", centroids.Group)

		if total == 0 {
			break
		}
	}

}

func (c *Centroids) Size() int {
	return len(c.Group)
}

func (c *Centroids) Reset(entries []PopulationEntry) {
	newCentroids := map[int]Centroid{}

	for k, v := range c.Group {
		count := 0
		sum := 0.0

		for _, ent := range entries {
			if ent.Group == k {
				count++
				sum += float64(ent.Count)
			}
		}
		log.Printf("Reset: for %v, fount %v entries", k, count)

		if count > 0 {
			newCentroids[k] = Centroid{Mean: sum / float64(count)}
		} else {
			log.Printf("Warning: Centroid[%v] = %v does not have any entry", k, v)
		}
	}

	c.Group = newCentroids
}

func (c *Centroids) Nearest(pop int) int {
	dist := math.MaxFloat64
	nearest_id := -1

	for k, v := range c.Group {
		d := math.Abs(v.Mean - float64(pop))

		if d < dist {
			dist = d
			nearest_id = k
		}
	}
	// log.Printf("%v is close to [%v]=%v among %v", pop, nearest_id, c.Group[nearest_id], c.Group)

	return nearest_id
}

func NewCentroidsOld(entries []PopulationEntry, ngroup int) *Centroids {
	centroids := map[int]Centroid{}

	prevCount := -1
	index := 0
	for _, ent := range entries {
		if prevCount != ent.Count {
			prevCount = ent.Count
			centroids[index] = Centroid{float64(ent.Count)}
			index++
		}
		if index >= ngroup {
			break
		}
	}

	return &Centroids{Group: centroids}
}

func NewCentroids(entries []PopulationEntry, ngroup int) *Centroids {
	centroids := map[int]Centroid{}

	largest := entries[0].Count
	unit := float64(largest) / float64(ngroup)

	for i := 0; i < ngroup; i++ {
		centroids[i] = Centroid{(unit + 1) * float64(i)}
	}

	return &Centroids{Group: centroids}
}

func (s *Server) serveStatistic(r StatisticRequest) {
	writer := NewPopulationWriter(r.Stream, r.Formatter)
	defer writer.Flush()

	writer.WriteHeader()
	entries := make([]PopulationEntry, 0, len(s.population))
	for _, v := range s.population {
		entries = append(entries, v)
	}
	sort.Sort(ByPopulation(entries))

	if r.Limit < 0 {
		r.Limit = len(entries)
	}
	if r.Limit > len(entries) {
		r.Limit = len(entries)
	}

	if r.Groups > 0 && len(s.population) >= r.Groups {
		Group(entries[:r.Limit], r.Groups, r.MaxGroupIteration)
	}

	for i := 0; i < r.Limit; i++ {
		writer.WriteEntry(entries[i])
	}

	close(r.Done)
}

func (s *Server) Close() error {
	close(s.quitChannel)
	for _, ln := range s.Listeners {
		ln.Close()
	}
	s.listenerGroup.Wait()
	s.workerGroup.Wait()

	close(s.Incoming)
	s.serverGroup.Wait()
	return nil
}

func (s *Server) AddListener(network string, laddr string) error {
	log.Printf("Listening on %v, %v", network, laddr)
	ln, err := net.Listen(network, laddr)
	if err != nil {
		return err
	}

	s.Listeners = append(s.Listeners, ln)
	s.listenerGroup.Add(1)

	go func() {
		defer s.listenerGroup.Done()
	loop:
		for {
			conn, err := ln.Accept()
			if err != nil {
				select {
				case <-s.quitChannel:
					break loop
				default:
					Err(0, err, "accept failed: %v", err)
					continue
				}
			}

			go s.createWorker(conn)
		}
	}()

	return nil
}

func IsTimeout(err error) bool {
	if e, ok := err.(net.Error); ok && e.Timeout() {
		return true
	} else {
		return false
	}
}

func (s *Server) createWorker(conn net.Conn) {
	s.workerGroup.Add(1)

	reader := bufio.NewReader(conn)

	go func() {
		defer s.workerGroup.Done()
		defer conn.Close()

	loop:
		for {
			conn.SetReadDeadline(time.Now().Add(time.Duration(READ_TIMEOUT_SECONDS) * time.Second))
			line, err := reader.ReadString('\n')

			if err != nil {
				if err != io.EOF && !IsTimeout(err) {
					Err(0, err, "read failed: %v", err)
				}
				break
			}

			cmd := strings.TrimSpace(line)
			if cmd == "" {
				continue
			}

			log.Printf("cmd[0]: [%T] %v", cmd[0], cmd[0])
			if cmd[0] != '!' && cmd[0] != '.' {
				resp := make(chan BlockEntry)
				s.Incoming <- LocationRequest{Address: cmd, Result: resp}
				result := <-resp

				if result.City.Country == "" {
					result.City.Country = "UNKNOWN"
				}
				if result.City.Name == "" {
					result.City.Name = "UNKNOWN"
				}

				msg := fmt.Sprintf("%v:%v\n", result.City.Country, result.City.Name)
				conn.Write([]byte(msg))
			} else {
				args := strings.Split(cmd[1:], " ")

				if len(args) <= 0 {
					continue
				}

				log.Printf("command %v received", args[0])
				switch strings.ToUpper(args[0]) {
				case "QUIT":
					break loop
				case "STAT":
					s.doStat(conn, args[1:])
				case "RESET":
					s.Incoming <- ResetRequest{}
				default:
					log.Printf("unrecognized command %v received", args[0])
				}
			}
		}
	}()
}

func (s *Server) doStat(conn net.Conn, args []string) error {
	var r StatisticRequest
	r.Formatter, _ = NewFormatter("csv", fieldOrder, fieldSeparator)
	r.Stream = conn
	r.Done = make(chan struct{})
	r.Limit = limitCount
	r.Groups = numGroups
	r.MaxGroupIteration = numGroupIteration

	for _, arg := range args {
		toks := strings.Split(arg, "=")

		var value string
		if len(toks) >= 2 {
			value = toks[1]
		}

		switch strings.ToUpper(toks[0]) {
		case "LIMIT":
			ival, err := strconv.ParseInt(value, 0, 64)
			if err != nil {
				return fmt.Errorf("cannot convert %v to int", value)
			}
			r.Limit = int(ival)
		case "GROUPS":
			ival, err := strconv.ParseInt(value, 0, 64)
			if err != nil {
				return fmt.Errorf("cannot convert %v to int", value)
			}
			if int(ival) < r.Groups {
				r.Groups = int(ival)
			}
		case "ITERATION":
			ival, err := strconv.ParseInt(value, 0, 64)
			if err != nil {
				return fmt.Errorf("cannot convert %v to int", value)
			}
			if int(ival) < r.MaxGroupIteration {
				r.MaxGroupIteration = int(ival)
			}
		case "FORMAT":
			formatter, err := NewFormatter(value, fieldOrder, fieldSeparator)
			if err != nil {
				return err
			}
			r.Formatter = formatter
		}
	}
	s.Incoming <- r
	<-r.Done
	return nil
}

func (s *Server) Start() {
	s.serverGroup.Add(1)

	go func() {
		defer s.serverGroup.Done()

		for request := range s.Incoming {
			switch r := request.(type) {
			case LocationRequest:
				// log.Printf("LOCATION request received: %v", r)
				s.serveLocation(r)
			case StatisticRequest:
				log.Printf("STAT request received: %v", r)
				s.serveStatistic(r)
			case ResetRequest:
				log.Printf("RESET request received")
				s.population = map[string]PopulationEntry{}
			}
		}
	}()
}
