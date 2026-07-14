package reporter

import (
	"bytes"
	"fmt"
	"io"

	"github.com/aarsakian/FileSystemForensics/utils"
)

// ANSI colors
const (
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	White  = "\033[37m"
	Bold   = "\033[1m"
	Reset  = "\033[0m"
)

type TableManager struct {
	Columns []Column
	W       io.Writer
}

type Column struct {
	Name  string
	Width int
}

func (tm *TableManager) DetermineColumnWidths(showFull, showFileSize, showPath, showClusters,
	showVCNs, showIndex, showParent, showReparse, showDeletion,
	showTimestamps, showRunLists, showFilename, IsResident, showUSNJRNL bool) {

	activeColumns := 0

	if showFull {
		tm.Columns = append(tm.Columns, Column{Name: "ID"})
		tm.Columns = append(tm.Columns, Column{Name: "Type"})
		activeColumns += 2
	}

	if showFilename || showUSNJRNL || showFull {
		tm.Columns = append(tm.Columns, Column{Name: "Filename"})
		activeColumns++
	}

	if showUSNJRNL {
		tm.Columns = append(tm.Columns, Column{Name: "Reason"})
		tm.Columns = append(tm.Columns, Column{Name: "File Attributes"})
		tm.Columns = append(tm.Columns, Column{Name: "Entry Reference:Sequence"})
		tm.Columns = append(tm.Columns, Column{Name: "Parent Reference:Sequence"})
		tm.Columns = append(tm.Columns, Column{Name: "Event Time"})

		activeColumns += 5
	}

	if showTimestamps || showFull {
		tm.Columns = append(tm.Columns, Column{Name: "FNA Access Time"})
		tm.Columns = append(tm.Columns, Column{Name: "FNA Creation Time"})
		tm.Columns = append(tm.Columns, Column{Name: "FNA Modification Time"})
		tm.Columns = append(tm.Columns, Column{Name: "FNA $MFT Modification Time"})
		activeColumns += 4

		tm.Columns = append(tm.Columns, Column{Name: "SI Access Time"})
		tm.Columns = append(tm.Columns, Column{Name: "SI Creation Time"})
		tm.Columns = append(tm.Columns, Column{Name: "SI Modification Time"})
		tm.Columns = append(tm.Columns, Column{Name: "SI $MFT Modification Time"})
		activeColumns += 4

		tm.Columns = append(tm.Columns, Column{Name: "$I30 Access Time"})
		tm.Columns = append(tm.Columns, Column{Name: "$I30 Creation Time"})
		tm.Columns = append(tm.Columns, Column{Name: "$I30 Modification Time"})
		tm.Columns = append(tm.Columns, Column{Name: "$I30 $MFT Modification Time"})
		activeColumns += 4
	}

	if IsResident || showFull {
		tm.Columns = append(tm.Columns, Column{Name: "Resident"})
		activeColumns++
	}

	if showFileSize || showFull {
		tm.Columns = append(tm.Columns, Column{Name: "Logical Size (KB)"})
		tm.Columns = append(tm.Columns, Column{Name: "Physical Size (KB)"})
		activeColumns += 2
	}
	if showParent || showFull {
		tm.Columns = append(tm.Columns, Column{Name: "Parent Info"})
		activeColumns++
	}
	if showPath || showFull {
		tm.Columns = append(tm.Columns, Column{Name: "Path"})
		activeColumns++
	}

	if showClusters || showFull {
		tm.Columns = append(tm.Columns, Column{Name: "Allocated Clusters"})
		activeColumns++
	}

	if showVCNs || showFull {
		tm.Columns = append(tm.Columns, Column{Name: "Start VCN:Last VCN"})
		activeColumns++
	}
	if showIndex {
		tm.Columns = append(tm.Columns, Column{Name: "Index"})
		activeColumns++
	}

	if showReparse {
		tm.Columns = append(tm.Columns, Column{Name: "Reparse Point"})
		activeColumns++
	}

	if showRunLists || showFull {
		tm.Columns = append(tm.Columns, Column{Name: "Runlist Cluster Offset:Cluster Length"})
		activeColumns++
	}

	if showDeletion {
		tm.Columns = append(tm.Columns, Column{Name: "Cluster:allocation status"})
		activeColumns++
	}

	totalWidth := utils.TerminalWidth() - 5
	if activeColumns > 0 {
		widthPerColumn := totalWidth / activeColumns
		for i := range tm.Columns {
			tm.Columns[i].Width = widthPerColumn
		}
	}

}

func (tm TableManager) WriteRow(parts ...string) error {
	var buf bytes.Buffer
	for _, p := range parts {
		buf.WriteString(p)
	}
	buf.WriteByte('\n')
	_, err := io.WriteString(tm.W, buf.String())
	return err
}

func (tm TableManager) PrintHeader() error {
	// Build top border
	var top bytes.Buffer
	top.WriteString("┌")
	for i, col := range tm.Columns {
		for j := 0; j < col.Width; j++ {
			top.WriteRune('─')
		}
		if i < len(tm.Columns)-1 {
			top.WriteString("┬")
		}
	}
	top.WriteString("┐")
	if err := tm.WriteRow(top.String()); err != nil {
		return err
	}

	// Header row
	var header bytes.Buffer
	header.WriteString("│")
	for _, col := range tm.Columns {
		header.WriteString(fmt.Sprintf(" %s%-*s%s ", Bold+Blue, col.Width-2, col.Name, Reset))
		header.WriteString("│")
	}
	if err := tm.WriteRow(header.String()); err != nil {
		return err
	}

	// Separator
	var sep bytes.Buffer
	sep.WriteString("├")
	for i, col := range tm.Columns {
		for j := 0; j < col.Width; j++ {
			sep.WriteRune('─')
		}
		if i < len(tm.Columns)-1 {
			sep.WriteString("┼")
		}
	}
	sep.WriteString("┤")
	if err := tm.WriteRow(sep.String()); err != nil {
		return err
	}
	return nil
}

func (tm TableManager) PrintFooter() error {
	// Bottom border
	var bottom bytes.Buffer
	bottom.WriteString("└")
	for i, col := range tm.Columns {
		for j := 0; j < col.Width; j++ {
			bottom.WriteRune('─')
		}
		if i < len(tm.Columns)-1 {
			bottom.WriteString("┴")
		}
	}
	bottom.WriteString("┘")
	return tm.WriteRow(bottom.String())
}

func (tm TableManager) PrintRow(row []string) error {

	// Data rows

	var line bytes.Buffer
	line.WriteString("│")
	for colidx, col := range tm.Columns {

		if len(row[colidx]) > col.Width-2 {
			row[colidx] = row[colidx][:col.Width-2] + "..."
		}
		if col.Name == "Type" && (row[colidx] == "Folder Unallocated" ||
			row[colidx] == "File Unallocated") {
			fmt.Fprintf(&line, "%s%-*s%s", Red, col.Width, row[colidx], Reset)
		} else {
			fmt.Fprintf(&line, "%s%-*s%s", White, col.Width, row[colidx], Reset)
		}

		line.WriteString("│")
	}
	if err := tm.WriteRow(line.String()); err != nil {
		return err
	}
	return nil

}
