package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

type Formatter interface {
	WriteHeader(writer *bufio.Writer) error
	WriteEntry(writer *bufio.Writer, entry PopulationEntry) error
}

type PopulationWriter struct {
	Formatter Formatter
	Writer    *bufio.Writer
	Stream    io.Writer
}

func NewPopulationWriter(out io.Writer, formatter Formatter) *PopulationWriter {
	return &PopulationWriter{
		Formatter: formatter,
		Stream:    out,
		Writer:    bufio.NewWriter(out),
	}
}

func (w *PopulationWriter) WriteHeader() error {
	return w.Formatter.WriteHeader(w.Writer)
}

func (w *PopulationWriter) WriteEntry(entry PopulationEntry) error {
	err := w.Formatter.WriteEntry(w.Writer, entry)
	return err
}

func (w *PopulationWriter) Flush() error {
	return w.Writer.Flush()
}

/*
func (w *PopulationWriter) Close() error {
	err := w.Writer.Flush()
	if err != nil {
		return err
	}
	return w.Stream.Close()
}
*/

type CSVFormatter struct {
	FieldOrder []PopulationField
}

func NewCSVFormatter(order []PopulationField) *CSVFormatter {
	return &CSVFormatter{FieldOrder: order}
}

func (f *CSVFormatter) WriteHeader(writer *bufio.Writer) error {
	names := make([]string, 0, len(f.FieldOrder))
	for _, f := range f.FieldOrder {
		names = append(names, PopulationFieldToName[f])
	}
	_, err := writer.WriteString(fmt.Sprintf("%v\n", strings.Join(names, ",")))
	return err
}

func (f *CSVFormatter) WriteEntry(writer *bufio.Writer, entry PopulationEntry) error {
	fields := make([]string, 0, len(f.FieldOrder))

	for _, f := range f.FieldOrder {
		switch f {
		case F_NAME:
			fields = append(fields, fmt.Sprintf("\"%v\"", entry.Name))
		case F_LONGITUDE:
			fields = append(fields, fmt.Sprintf("%v", entry.Longitude))
		case F_LATITUDE:
			fields = append(fields, fmt.Sprintf("%v", entry.Latitude))
		case F_COUNT:
			fields = append(fields, fmt.Sprintf("%v", entry.Count))
		case F_GROUP:
			fields = append(fields, fmt.Sprintf("%v", entry.Group))
		}
	}
	_, err := writer.WriteString(fmt.Sprintf("%v\n", strings.Join(fields, ",")))
	return err
}

type TextFormatter struct {
	FieldSeparator string
	FieldOrder     []PopulationField
}

func NewTextFormatter(order []PopulationField, sep string) *TextFormatter {
	return &TextFormatter{FieldSeparator: sep, FieldOrder: order}
}

func (f *TextFormatter) WriteHeader(writer *bufio.Writer) error {
	names := make([]string, 0, len(f.FieldOrder))
	for _, f := range f.FieldOrder {
		names = append(names, PopulationFieldToName[f])
	}
	_, err := writer.WriteString(fmt.Sprintf("%v\n", strings.Join(names, f.FieldSeparator)))
	return err
}

func (f *TextFormatter) WriteEntry(writer *bufio.Writer, entry PopulationEntry) error {
	fields := make([]string, 0, len(f.FieldOrder))

	for _, f := range f.FieldOrder {
		switch f {
		case F_NAME:
			fields = append(fields, fmt.Sprintf("%v", entry.Name))
		case F_LONGITUDE:
			fields = append(fields, fmt.Sprintf("%v", entry.Longitude))
		case F_LATITUDE:
			fields = append(fields, fmt.Sprintf("%v", entry.Latitude))
		case F_COUNT:
			fields = append(fields, fmt.Sprintf("%v", entry.Count))
		case F_GROUP:
			fields = append(fields, fmt.Sprintf("%v", entry.Group))
		}
	}
	_, err := writer.WriteString(fmt.Sprintf("%v\n", strings.Join(fields, f.FieldSeparator)))
	return err
}

func NewFormatter(formatType string, fieldOrder string, fieldSeparator string) (Formatter, error) {
	forder, err := ParseFieldOrder(fieldOrder)
	if err != nil {
		return nil, err
	}

	switch formatType {
	case "text":
		return NewTextFormatter(forder, fieldSeparator), nil
	case "txt":
		return NewTextFormatter(forder, fieldSeparator), nil
	case "csv":
		return NewCSVFormatter(forder), nil
	default:
		return nil, fmt.Errorf("unknown formatter type: %v", formatterName)
	}
}
