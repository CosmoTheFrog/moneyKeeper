// Package xlsxread читает листы xlsx как таблицы строк, без внешних зависимостей. Формулы не считаются,
// берутся сохранённые значения. Даты остаются числами Excel, перевод в день делает вызывающий.
package xlsxread

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
)

const maxPartBytes = 20 << 20

var ErrInvalid = errors.New("invalid xlsx")

type Book struct {
	zip     *zip.Reader
	strings []string
	sheets  map[string]string // название листа → путь к файлу в архиве
}

func Open(data []byte) (*Book, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalid, err)
	}
	b := &Book{zip: zr, sheets: map[string]string{}}

	var wb struct {
		Sheets []struct {
			Name string `xml:"name,attr"`
			RID  string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
		} `xml:"sheets>sheet"`
	}
	if err := b.decode("xl/workbook.xml", &wb); err != nil {
		return nil, err
	}
	var rels struct {
		Items []struct {
			ID     string `xml:"Id,attr"`
			Target string `xml:"Target,attr"`
		} `xml:"Relationship"`
	}
	if err := b.decode("xl/_rels/workbook.xml.rels", &rels); err != nil {
		return nil, err
	}
	target := map[string]string{}
	for _, r := range rels.Items {
		target[r.ID] = strings.TrimPrefix(path.Join("xl", r.Target), "/")
	}
	for _, s := range wb.Sheets {
		b.sheets[s.Name] = target[s.RID]
	}

	// Таблицы строк может не быть, если все ячейки числовые.
	var sst struct {
		Items []struct {
			Text []string `xml:"t"`
			Runs []struct {
				Text string `xml:"t"`
			} `xml:"r"`
		} `xml:"si"`
	}
	if err := b.decode("xl/sharedStrings.xml", &sst); err == nil {
		for _, si := range sst.Items {
			var sb strings.Builder
			for _, t := range si.Text {
				sb.WriteString(t)
			}
			for _, r := range si.Runs {
				sb.WriteString(r.Text)
			}
			b.strings = append(b.strings, sb.String())
		}
	}
	return b, nil
}

// Sheet возвращает строки листа, ячейки по номерам колонок, пустые как "". ok=false, если листа с таким именем нет.
func (b *Book) Sheet(name string) (rows [][]string, ok bool, err error) {
	file, ok := b.sheets[name]
	if !ok {
		return nil, false, nil
	}
	var ws struct {
		Rows []struct {
			Cells []struct {
				Ref    string `xml:"r,attr"`
				Type   string `xml:"t,attr"`
				Value  string `xml:"v"`
				Inline string `xml:"is>t"`
			} `xml:"c"`
		} `xml:"sheetData>row"`
	}
	if err := b.decode(file, &ws); err != nil {
		return nil, true, err
	}
	for _, r := range ws.Rows {
		var row []string
		for _, c := range r.Cells {
			col := columnIndex(c.Ref)
			for len(row) <= col {
				row = append(row, "")
			}
			switch c.Type {
			case "s":
				var i int
				if _, err := fmt.Sscanf(c.Value, "%d", &i); err != nil || i < 0 || i >= len(b.strings) {
					return nil, true, fmt.Errorf("%w: shared string %q", ErrInvalid, c.Value)
				}
				row[col] = b.strings[i]
			case "inlineStr":
				row[col] = c.Inline
			default:
				row[col] = c.Value
			}
		}
		rows = append(rows, row)
	}
	return rows, true, nil
}

func (b *Book) decode(name string, dst any) error {
	for _, f := range b.zip.File {
		if f.Name != name {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("%w: %w", ErrInvalid, err)
		}
		defer rc.Close()
		if err := xml.NewDecoder(io.LimitReader(rc, maxPartBytes)).Decode(dst); err != nil {
			return fmt.Errorf("%w: %s: %w", ErrInvalid, name, err)
		}
		return nil
	}
	return fmt.Errorf("%w: no %s", ErrInvalid, name)
}

// columnIndex превращает ссылку вида «AB12» в номер колонки с нуля.
func columnIndex(ref string) int {
	n := 0
	for _, r := range ref {
		if r < 'A' || r > 'Z' {
			break
		}
		n = n*26 + int(r-'A') + 1
	}
	return n - 1
}
