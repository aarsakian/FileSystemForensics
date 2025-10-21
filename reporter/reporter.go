package reporter

import (
	"fmt"

	metadata "github.com/aarsakian/FileSystemForensics/FS"
	UsnJrnl "github.com/aarsakian/FileSystemForensics/FS/NTFS/usnjrnl"
	"github.com/aarsakian/FileSystemForensics/tree"
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
}

func (rp Reporter) Show(records []metadata.Record, usnjrnlRecords UsnJrnl.Records, partitionId int, tree tree.Tree) {
	for _, record := range records {

		if record.GetID() == 0 {
			continue
		}

		fmt.Printf("%d  --------------------------------------------------------------------\n", record.GetID())

		if rp.ShowFileName != "" {
			record.ShowAttributes("FileName")

		}

		if rp.ShowAttributes != "" {
			record.ShowAttributes(rp.ShowAttributes)

		}

		if rp.ShowTimestamps {
			record.ShowTimestamps()

		}

		if rp.IsResident {
			record.ShowIsResident()

		}

		if rp.ShowRunList {
			record.ShowRunList()

		}

		if rp.ShowFileSize {
			record.ShowFileSize()

		}

		if rp.ShowVCNs {
			record.ShowVCNs()

		}

		if rp.ShowIndex {

			record.ShowIndex()

		}

		if rp.ShowParent {
			record.ShowParentRecordInfo()
		}

		if rp.ShowPath {
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
