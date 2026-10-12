package spreadsheet

import (
	"context"
	"fmt"

	"github.com/spf13/cast"
	"google.golang.org/api/sheets/v4"
)

type Col any

type Row struct {
	ColIndex map[string]int
	Data     []Col
	Index    int
}

type Sheet struct {
	ColIndex map[string]int
	Rows     []Row
	Cols     []string
}

func (r Row) Get(name string) Col {
	index, ok := r.ColIndex[name]
	if !ok {
		return nil
	}
	if index == -1 || index >= len(r.Data) {
		return nil
	}
	return r.Data[index]
}

func GetSheet(ctx context.Context, service *sheets.Service, spreadsheetID string, sheet string) (*Sheet, error) {
	res, err := service.Spreadsheets.Values.Get(spreadsheetID, sheet+"!A:Z").Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("find spreadsheet %s rows in sheet %s failed: %w", spreadsheetID, sheet, err)
	}
	if len(res.Values) == 0 {
		return nil, fmt.Errorf("spreadsheet %s rows in sheet %s are empty", spreadsheetID, sheet)
	}
	var cols []string
	colIndex := make(map[string]int)
	rows := make([]Row, len(res.Values))
	for i, row := range res.Values {
		if i == 0 {
			for j, col := range row {
				name := cast.ToString(col)
				colIndex[name] = j
				cols = append(cols, name)
			}
			continue
		}
		rows[i].Index = i + 1
		rows[i].ColIndex = colIndex
		rows[i].Data = make([]Col, len(row))
		for j, value := range row {
			col, ok := value.(Col)
			if !ok {
				return nil, fmt.Errorf(
					"find column %s rows in sheet %s: invalid type in row[%d][%d]", spreadsheetID, sheet, i, j,
				)
			}
			rows[i].Data[j] = col
		}
	}
	return &Sheet{
		Rows:     rows[1:],
		ColIndex: colIndex,
		Cols:     cols,
	}, nil
}

func AppendRows(
	ctx context.Context, service *sheets.Service, spreadsheetID string, sheet string, columns []string,
	rows []map[string]any,
) error {
	values := make([][]any, len(rows))
	for i, row := range rows {
		values[i] = make([]any, len(columns))
		for j, col := range columns {
			values[i][j] = row[col]
		}
	}
	if _, err := service.Spreadsheets.Values.
		Append(
			spreadsheetID, sheet, &sheets.ValueRange{
				Values: values,
			},
		).
		ValueInputOption("RAW").
		InsertDataOption("INSERT_ROWS").
		Context(ctx).
		Do(); err != nil {
		return fmt.Errorf("append row to spreadsheet %s sheet %s failed: %w", spreadsheetID, sheet, err)
	}
	return nil
}

func UpdateRows(
	ctx context.Context, service *sheets.Service, spreadsheetID string, sheet string, columns []string,
	updatedRows []map[string]any, rowIndex map[string]int, getRowKey func(row map[string]any) string,
) error {
	data := make([]*sheets.ValueRange, 0, len(updatedRows))
	for _, updatedRow := range updatedRows {
		id := getRowKey(updatedRow)
		index, hasIndex := rowIndex[id]
		if !hasIndex {
			continue
		}
		vr := &sheets.ValueRange{
			// +1 because rowIndex is 0-based, +1 for header row
			Range: fmt.Sprintf("%s!A%d", sheet, index),
			Values: [][]any{
				make([]any, len(columns)),
			},
		}
		for j, col := range columns {
			v, ok := updatedRow[col]
			if !ok {
				v = ""
			}
			vr.Values[0][j] = v
		}
		data = append(data, vr)
	}
	if len(data) == 0 {
		return nil
	}
	if _, err := service.Spreadsheets.Values.
		BatchUpdate(
			spreadsheetID, &sheets.BatchUpdateValuesRequest{
				ValueInputOption: "RAW",
				Data:             data,
			},
		).
		Context(ctx).
		Do(); err != nil {
		return fmt.Errorf("update row in spreadsheet %s sheet %s failed: %w", spreadsheetID, sheet, err)
	}
	return nil
}
