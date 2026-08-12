package mintegral

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
)

// ReportStream 逐行读取报表；Close 可安全重复调用。
type ReportStream struct {
	body      io.ReadCloser
	reader    *csv.Reader
	decoder   reportRowDecoder
	initErr   error
	closeOnce sync.Once
	closeErr  error
	status    ReportStatus
}

func newReportStream(body io.ReadCloser, source io.Reader) *ReportStream {
	reader := csv.NewReader(source)
	reader.Comma = '\t'
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err == nil && len(header) > 0 {
		header[0] = strings.TrimPrefix(header[0], "\ufeff")
	}
	decoder, decodeErr := newReportRowDecoder(header)
	if err == nil {
		err = decodeErr
	}
	return &ReportStream{body: body, reader: reader, decoder: decoder, initErr: err}
}

// Next 返回下一行；流结束时返回 io.EOF。
func (s *ReportStream) Next() (ReportRow, error) {
	if s == nil {
		return ReportRow{}, io.ErrClosedPipe
	}
	if s.initErr != nil {
		err := s.initErr
		s.initErr = nil
		return ReportRow{}, fmt.Errorf("%w: TSV header: %w", ErrInvalidReport, err)
	}
	record, err := s.reader.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return ReportRow{}, io.EOF
		}
		return ReportRow{}, fmt.Errorf("%w: TSV record: %w", ErrInvalidReport, err)
	}
	return s.decoder.decode(record)
}

// Close 关闭下载响应体；重复调用返回相同结果。
func (s *ReportStream) Close() error {
	if s == nil {
		return nil
	}
	s.closeOnce.Do(func() { s.closeErr = s.body.Close() })
	return s.closeErr
}

type reportColumn struct {
	name string
	set  func(*ReportRow, string) error
}
type reportRowDecoder struct {
	columns    []reportColumn
	extraCount int
}

func newReportRowDecoder(header []string) (reportRowDecoder, error) {
	if len(header) == 0 {
		return reportRowDecoder{}, errors.New("empty header")
	}
	decoder := reportRowDecoder{columns: make([]reportColumn, len(header))}
	seen := make(map[string]struct{}, len(header))
	for index, name := range header {
		name = strings.TrimSpace(name)
		if name == "" {
			return reportRowDecoder{}, errors.New("blank column")
		}
		if _, exists := seen[name]; exists {
			return reportRowDecoder{}, fmt.Errorf("duplicate column %q", name)
		}
		seen[name] = struct{}{}
		setter := reportSetter(name)
		if setter == nil {
			decoder.extraCount++
		}
		decoder.columns[index] = reportColumn{name: name, set: setter}
	}
	for _, required := range []string{"Date", "Currency", "Impression", "Click", "Conversion", "Ecpm", "Cpc", "Ctr", "Cvr", "Ivr", "Spend"} {
		if _, exists := seen[required]; !exists {
			return reportRowDecoder{}, fmt.Errorf("missing column %q", required)
		}
	}
	return decoder, nil
}

func (d reportRowDecoder) decode(record []string) (ReportRow, error) {
	if len(record) != len(d.columns) {
		return ReportRow{}, fmt.Errorf("%w: TSV row has %d columns, expected %d", ErrInvalidReport, len(record), len(d.columns))
	}
	row := ReportRow{}
	if d.extraCount > 0 {
		row.Extras.values = make(map[string]string, d.extraCount)
	}
	for index, column := range d.columns {
		if column.set == nil {
			row.Extras.values[column.name] = record[index]
			continue
		}
		if err := column.set(&row, record[index]); err != nil {
			return ReportRow{}, fmt.Errorf("%w: column %q: %w", ErrInvalidReport, column.name, err)
		}
	}
	return row, nil
}

func parseRowInt(field func(*ReportRow) *int64) func(*ReportRow, string) error {
	return func(row *ReportRow, value string) error {
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err == nil {
			*field(row) = parsed
		}
		return err
	}
}

func parseRowDecimal(field func(*ReportRow) *DecimalText) func(*ReportRow, string) error {
	return func(row *ReportRow, value string) error {
		parsed, err := ParseDecimalText(value)
		if err == nil {
			*field(row) = parsed
		}
		return err
	}
}
