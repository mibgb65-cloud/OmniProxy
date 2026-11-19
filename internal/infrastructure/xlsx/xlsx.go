package xlsx

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	contentTypesPart = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
		`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
		`<Default Extension="xml" ContentType="application/xml"/>` +
		`<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>` +
		`<Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>` +
		`</Types>`
	rootRelationships = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
		`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>` +
		`</Relationships>`
	workbookRelationships = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
		`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>` +
		`</Relationships>`
	worksheetOpen = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`
	worksheetClose = `</sheetData></worksheet>`
)

// Write 创建一份单工作表 XLSX；writeRows 通过 yield 逐行提供数据。
// 调用方负责决定单元格内容（包括按 FitCell 截断），这里只按当前行号写入。
// 先写临时文件再替换目标，失败时保留用户已存在的文件。
func Write(path, sheetName string, headers []string, writeRows func(yield func([]Cell) error) error) (int, error) {
	if !filepath.IsAbs(path) || !strings.EqualFold(filepath.Ext(path), ".xlsx") {
		return 0, errors.New("xlsx: invalid_path")
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".omniproxy-xlsx-*.tmp")
	if err != nil {
		return 0, errors.New("xlsx: export_failed")
	}
	temporary := file.Name()
	defer func() { _ = os.Remove(temporary) }()
	archive := zip.NewWriter(file)
	if err := writeParts(archive, sheetName); err != nil {
		return 0, finishError(archive, file)
	}
	sheet, err := archive.Create("xl/worksheets/sheet1.xml")
	if err != nil {
		return 0, finishError(archive, file)
	}
	rows, err := writeSheet(sheet, headers, writeRows)
	if err != nil {
		return 0, finishError(archive, file)
	}
	if err := archive.Close(); err != nil {
		_ = file.Close()
		return 0, errors.New("xlsx: export_failed")
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return 0, errors.New("xlsx: export_failed")
	}
	if err := file.Close(); err != nil {
		return 0, errors.New("xlsx: export_failed")
	}
	if err := os.Rename(temporary, path); err != nil {
		return 0, errors.New("xlsx: export_failed")
	}
	return rows, nil
}

func finishError(archive *zip.Writer, file *os.File) error {
	_ = archive.Close()
	_ = file.Close()
	return errors.New("xlsx: export_failed")
}

// writeParts 写入工作簿需要的固定包结构；工作表正文随后流式写入。
func writeParts(archive *zip.Writer, sheetName string) error {
	parts := []struct{ name, body string }{
		{"[Content_Types].xml", contentTypesPart},
		{"_rels/.rels", rootRelationships},
		{"xl/_rels/workbook.xml.rels", workbookRelationships},
	}
	for _, part := range parts {
		writer, err := archive.Create(part.name)
		if err != nil {
			return err
		}
		if _, err := io.WriteString(writer, part.body); err != nil {
			return err
		}
	}
	writer, err := archive.Create("xl/workbook.xml")
	if err != nil {
		return err
	}
	_, err = io.WriteString(writer, workbookXML(sanitizeSheetName(sheetName)))
	return err
}

func workbookXML(sheetName string) string {
	name := xmlEscape(sheetName)
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">` +
		`<sheets><sheet name="` + name + `" sheetId="1" r:id="rId1"/></sheets></workbook>`
}

func writeSheet(sheet io.Writer, headers []string, writeRows func(yield func([]Cell) error) error) (int, error) {
	if _, err := io.WriteString(sheet, worksheetOpen); err != nil {
		return 0, err
	}
	cells := make([]Cell, len(headers))
	for index, header := range headers {
		cells[index] = Cell{Text: header}
	}
	if err := writeRow(sheet, 1, cells); err != nil {
		return 0, err
	}
	row := 2
	if err := writeRows(func(cells []Cell) error {
		if err := writeRow(sheet, row, cells); err != nil {
			return err
		}
		row++
		return nil
	}); err != nil {
		return 0, err
	}
	if _, err := io.WriteString(sheet, worksheetClose); err != nil {
		return 0, err
	}
	return row - 2, nil
}

func writeRow(sheet io.Writer, row int, cells []Cell) error {
	if _, err := fmt.Fprintf(sheet, `<row r="%d">`, row); err != nil {
		return err
	}
	for index, cell := range cells {
		reference := columnName(index) + strconv.Itoa(row)
		switch {
		case cell.Number != "":
			if !validNumber(cell.Number) {
				return errors.New("xlsx: invalid_number")
			}
			if _, err := fmt.Fprintf(sheet, `<c r="%s"><v>%s</v></c>`, reference, strings.TrimSpace(cell.Number)); err != nil {
				return err
			}
		case cell.Text == "":
			continue
		default:
			text, _ := FitCell(cell.Text)
			if _, err := fmt.Fprintf(sheet, `<c r="%s" t="inlineStr"><is><t xml:space="preserve">`, reference); err != nil {
				return err
			}
			if _, err := io.WriteString(sheet, xmlEscape(sanitizeText(text))); err != nil {
				return err
			}
			if _, err := io.WriteString(sheet, `</t></is></c>`); err != nil {
				return err
			}
		}
	}
	_, err := io.WriteString(sheet, `</row>`)
	return err
}

// xmlEscape 转义 XML 文本；非法字符由调用方先清理。
func xmlEscape(text string) string {
	var builder strings.Builder
	_ = xml.EscapeText(&builder, []byte(text))
	return builder.String()
}
