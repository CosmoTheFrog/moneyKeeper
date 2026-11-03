package xlsxread

import (
	"archive/zip"
	"bytes"
	"errors"
	"reflect"
	"testing"
)

func buildXLSX(t *testing.T, parts map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range parts {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

const (
	workbook = `<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Data" sheetId="1" r:id="rId7"/></sheets></workbook>`
	rels     = `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId7" Target="worksheets/sheet1.xml"/></Relationships>`
	shared   = `<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><si><t>Date</t></si><si><r><t>Cof</t></r><r><t>fee</t></r></si></sst>`
	sheet    = `<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>
<row r="1"><c r="A1" t="s"><v>0</v></c><c r="C1" t="inlineStr"><is><t>Inline</t></is></c></row>
<row r="2"><c r="A2"><v>46296</v></c><c r="B2" t="s"><v>1</v></c></row>
</sheetData></worksheet>`
)

func TestSheet(t *testing.T) {
	data := buildXLSX(t, map[string]string{
		"xl/workbook.xml": workbook, "xl/_rels/workbook.xml.rels": rels,
		"xl/sharedStrings.xml": shared, "xl/worksheets/sheet1.xml": sheet,
	})
	b, err := Open(data)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		sheet  string
		want   [][]string
		wantOK bool
	}{
		{"cells by column, shared and inline strings", "Data", [][]string{{"Date", "", "Inline"}, {"46296", "Coffee"}}, true},
		{"missing sheet", "Nope", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok, err := b.Sheet(tt.sheet)
			if err != nil || ok != tt.wantOK {
				t.Fatalf("ok = %v err = %v, want ok %v", ok, err, tt.wantOK)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("rows = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOpenInvalid(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"not a zip", []byte("a,b\n1,2")},
		{"zip without workbook", buildXLSX(t, map[string]string{"x.txt": "hi"})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Open(tt.data); !errors.Is(err, ErrInvalid) {
				t.Errorf("err = %v, want ErrInvalid", err)
			}
		})
	}
}

func TestColumnIndex(t *testing.T) {
	for ref, want := range map[string]int{"A1": 0, "B7": 1, "Z3": 25, "AA1": 26, "AB10": 27} {
		if got := columnIndex(ref); got != want {
			t.Errorf("columnIndex(%q) = %d, want %d", ref, got, want)
		}
	}
}
