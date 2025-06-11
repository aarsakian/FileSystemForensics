package reporter

import (
	"fmt"

	metadata "github.com/aarsakian/FileSystemForensics/FS"
	UsnJrnl "github.com/aarsakian/FileSystemForensics/FS/NTFS/usnjrnl"
	"github.com/aarsakian/FileSystemForensics/tree"
)

type Reporter struct {
	ShowFileName   string
	ShowAttributes string
	ShowTimestamps bool
	IsResident     bool
	ShowFull       bool
	ShowRunList    bool
	ShowFileSize   bool
	ShowVCNs       bool
	ShowIndex      bool
	ShowParent     bool
	ShowPath       bool
	ShowUSNJRNL    bool
	ShowReparse    bool
	ShowTree       bool
}

func (rp Reporter) Show(records []metadata.Record, usnjrnlRecords UsnJrnl.Records, partitionId int, tree tree.Tree) {
	for _, record := range records {
		askedToShow := false
		if record.GetID() == 0 {
			continue
		}
		if rp.ShowFileName != "" || rp.ShowFull {
			record.ShowAttributes("FileName")
			askedToShow = true
		}

		if rp.ShowAttributes != "" || rp.ShowFull {
			record.ShowAttributes(rp.ShowAttributes)
			askedToShow = true
		}

		if rp.ShowTimestamps || rp.ShowFull {
			record.ShowTimestamps()
			askedToShow = true
		}

		if rp.IsResident || rp.ShowFull {
			record.ShowIsResident()
			askedToShow = true
		}

		if rp.ShowRunList || rp.ShowFull {
			record.ShowRunList()
			askedToShow = true
		}

		if rp.ShowFileSize || rp.ShowFull {
			record.ShowFileSize()
			askedToShow = true
		}

		if rp.ShowVCNs || rp.ShowFull {
			record.ShowVCNs()
			askedToShow = true
		}

		if rp.ShowIndex || rp.ShowFull {

			record.ShowIndex()
			askedToShow = true
		}

		if rp.ShowParent || rp.ShowFull {
			record.ShowParentRecordInfo()
		}

		if rp.ShowPath || rp.ShowFull {
			record.ShowPath(partitionId)
		}

		if rp.ShowReparse {
			record.ShowAttributes("Reparse Point")
		}

		if askedToShow {
			fmt.Printf("\n")
		}

	}

	for _, record := range usnjrnlRecords {
		if rp.ShowUSNJRNL {
			record.ShowInfo()
		}
	}

	if rp.ShowTree {
		tree.Show()
	}

}
