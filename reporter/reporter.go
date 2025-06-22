package reporter

import (
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

		if record.GetID() == 0 {
			continue
		}
		record.ShowInfo()

		if rp.ShowFileName != "" || rp.ShowFull {
			record.ShowAttributes("FileName")

		}

		if rp.ShowAttributes != "" || rp.ShowFull {
			record.ShowAttributes(rp.ShowAttributes)

		}

		if rp.ShowTimestamps || rp.ShowFull {
			record.ShowTimestamps()

		}

		if rp.IsResident || rp.ShowFull {
			record.ShowIsResident()

		}

		if rp.ShowRunList || rp.ShowFull {
			record.ShowRunList()

		}

		if rp.ShowFileSize || rp.ShowFull {
			record.ShowFileSize()

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
			record.ShowPath(partitionId)
		}

		if rp.ShowReparse {
			record.ShowAttributes("Reparse Point")
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
