package reporter

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	metadata "github.com/aarsakian/FileSystemForensics/FS"
	VssLib "github.com/aarsakian/FileSystemForensics/FS/NTFS/VSS"
	UsnJrnl "github.com/aarsakian/FileSystemForensics/FS/NTFS/usnjrnl"
	"github.com/aarsakian/FileSystemForensics/tree"
	"github.com/aarsakian/FileSystemForensics/utils"
)

type Reporter struct {
	ShowFileName    string
	ShowAttributes  string
	ShowTimestamps  bool
	IsResident      bool
	ShowFull        bool
	ShowRunList     bool
	ShowFileSize    bool
	ShowVCNs        bool
	ShowIndex       bool
	ShowParent      bool
	ShowPath        bool
	ShowUSNJRNL     bool
	ShowReparse     bool
	ShowTree        bool
	ShowVSSClusters bool
	ShowClusters    bool
	ShowDeletion    bool
	UseColor        bool
	HTMLReportPath  string
	Writer          io.Writer
	HTMLWriter      io.Writer
}

func (rp Reporter) Show(records []metadata.Record, usnjrnlRecords UsnJrnl.Records, partitionId int,
	tree tree.Tree, shadowVol *VssLib.ShadowVolume, clustersBitMap map[int]bool) {
	if rp.Writer == nil {
		rp.Writer = os.Stdout
	}
	if rp.UseColor || rp.HTMLWriter != nil {
		_ = rp.renderSummary(records, partitionId, "FileSystemForensics")
	}

	tm := TableManager{W: os.Stdout}
	tm.DetermineColumnWidths(rp.ShowFileSize, rp.ShowPath, rp.ShowClusters,
		rp.ShowVCNs, rp.ShowIndex, rp.ShowParent, rp.ShowReparse, rp.ShowDeletion,
		rp.ShowTimestamps)
	tm.PrintHeader()

	for _, record := range records {
		var vals []string
		if record.GetID() == 0 {
			continue
		}

		if rp.ShowFull {
			fmt.Println("-------------------------------------------------------")
			record.ShowInfo()
		}

		if rp.ShowFileName != "" || rp.ShowFull {
			record.ShowAttributes("FileName")
		}

		if rp.ShowAttributes != "" || rp.ShowFull {
			record.ShowAttributes(rp.ShowAttributes)
		}

		if rp.ShowTimestamps || rp.ShowFull {
			vals = append(vals, record.GetTimestamps()...)
			print(len(vals))

		}

		if rp.IsResident || rp.ShowFull {
			record.ShowIsResident()
		}

		if rp.ShowRunList || rp.ShowFull {
			record.ShowRunList()
		}

		if rp.ShowFileSize || rp.ShowFull {
			logicalsize, physicalsize := record.GetFileSize()
			vals = append(vals, fmt.Sprintf("%1.0f", float64(logicalsize)/1024))
			vals = append(vals, fmt.Sprintf("%1.0f", float64(physicalsize)/1024))
		}

		if rp.ShowVCNs || rp.ShowFull {
			record.ShowVCNs()
		}

		if rp.ShowIndex || rp.ShowFull {
			record.ShowIndex()
		}

		if rp.ShowParent || rp.ShowFull {
			record.ShowParentRecordInfo()
		}

		if rp.ShowPath || rp.ShowFull {
			vals = append(vals, record.GetPath(partitionId))
		}

		if rp.ShowClusters || rp.ShowFull {
			vals = append(vals, utils.ToString(record.GetAllocatedClusters()))
		}

		if rp.ShowReparse || rp.ShowFull {
			record.ShowAttributes("Reparse Point")
		}

		if rp.ShowVSSClusters && rp.ShowFull {
			clusters := record.GetAllocatedClusters()

			for idx, offset := range shadowVol.GetClustersInfo(4096, clusters) {
				if offset == -1 {
					fmt.Printf("%d cl shadow offset not found\n", clusters[idx])
				} else {
					fmt.Printf("%d cl shadow offset cl %d\n", clusters[idx], offset)
				}
			}
		}

		if rp.ShowDeletion || rp.ShowFull {
			record.ShowDeletionInfo(clustersBitMap)
		}
		tm.PrintRow(vals)

	}

	tm.PrintFooter()

	for _, record := range usnjrnlRecords {
		if rp.ShowUSNJRNL {
			record.ShowInfo()
		}
	}

	if rp.ShowTree {
		tree.Show()
	}

	if rp.HTMLWriter == nil && rp.HTMLReportPath != "" {
		file, err := os.Create(rp.HTMLReportPath)
		if err == nil {
			defer file.Close()
			rp.HTMLWriter = file
			_ = rp.renderHTMLReport(records, partitionId, "FileSystemForensics")
		}
	}

}

func (rp Reporter) renderSummary(records []metadata.Record, partitionID int, title string) error {
	var terminal bytes.Buffer
	fmt.Fprintf(&terminal, "%s\n", rp.formatHeading(title, partitionID))
	for _, record := range records {
		if record == nil {
			continue
		}
		fmt.Fprintf(&terminal, "● %s (ID %d)\n", rp.safeText(record.GetFname()), record.GetID())
	}
	_, err := io.WriteString(rp.Writer, terminal.String())
	return err
}

func (rp Reporter) renderHTMLReport(records []metadata.Record, partitionID int, title string) error {
	if rp.HTMLWriter == nil {
		return nil
	}
	var b strings.Builder
	b.WriteString("<!doctype html><html><head><meta charset='utf-8'><title>")
	b.WriteString(escapeHTML(title))
	b.WriteString("</title><style>body{font-family:Arial,sans-serif;margin:2rem;} .card{padding:1rem;border:1px solid #ddd;border-radius:8px;margin-bottom:1rem;}</style></head><body>")
	b.WriteString("<h1>")
	b.WriteString(escapeHTML(title))
	b.WriteString("</h1><p>Partition ")
	b.WriteString(fmt.Sprintf("%d", partitionID))
	b.WriteString(" • Generated ")
	b.WriteString(time.Now().UTC().Format(time.RFC3339))
	b.WriteString("</p>")
	for _, record := range records {
		if record == nil {
			continue
		}
		b.WriteString("<div class='card'><strong>")
		b.WriteString(escapeHTML(rp.safeText(record.GetFname())))
		b.WriteString("</strong><br/>ID: ")
		b.WriteString(fmt.Sprintf("%d", record.GetID()))
		b.WriteString("</div>")
	}
	b.WriteString("</body></html>")
	_, err := io.WriteString(rp.HTMLWriter, b.String())
	return err
}

func (rp Reporter) formatHeading(title string, partitionID int) string {
	if rp.UseColor {
		return fmt.Sprintf("\033[1;36m[%s]\033[0m partition=%d", title, partitionID)
	}
	return fmt.Sprintf("[%s] partition=%d", title, partitionID)
}

func (rp Reporter) safeText(value string) string {
	if value == "" {
		return "<unnamed>"
	}
	return strings.TrimSpace(value)
}

func escapeHTML(input string) string {
	input = strings.ReplaceAll(input, "&", "&amp;")
	input = strings.ReplaceAll(input, "<", "&lt;")
	input = strings.ReplaceAll(input, ">", "&gt;")
	input = strings.ReplaceAll(input, "\"", "&quot;")
	return strings.ReplaceAll(input, "'", "&#39;")
}
