package main

import (
	"fmt"
	"strings"
)

type PopulationField int

const (
	F_NAME PopulationField = iota
	F_LATITUDE
	F_LONGITUDE
	F_COUNT
	F_GROUP
)

type PopulationEntry struct {
	Name      string
	Latitude  float32
	Longitude float32
	Count     int
	Group     int
}

type ByPopulation []PopulationEntry

func (p ByPopulation) Len() int           { return len(p) }
func (p ByPopulation) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p ByPopulation) Less(i, j int) bool { return p[i].Count > p[j].Count }

var nameToPopulationField = map[string]PopulationField{
	"name":       F_NAME,
	"latitude":   F_LATITUDE,
	"longitude":  F_LONGITUDE,
	"lat":        F_LATITUDE,
	"lon":        F_LONGITUDE,
	"count":      F_COUNT,
	"pop":        F_COUNT,
	"population": F_COUNT,
	"group":      F_GROUP,
	"grp":        F_GROUP,
}

var PopulationFieldToName = map[PopulationField]string{
	F_NAME:      "name",
	F_LATITUDE:  "lat",
	F_LONGITUDE: "lon",
	F_COUNT:     "pop",
	F_GROUP:     "group",
}

func ParseFieldOrder(forder string) ([]PopulationField, error) {
	fnames := strings.Split(forder, ",")
	fids := make([]PopulationField, 0, len(fnames))

	for _, name := range fnames {
		nam := strings.TrimSpace(name)
		id, ok := nameToPopulationField[nam]

		if !ok {
			return nil, fmt.Errorf("field name '%v' not found", nam)
		}
		fids = append(fids, id)
	}
	return fids, nil
}
