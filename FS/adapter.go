package metadata

import (
	fstree "github.com/aarsakian/FileSystemForensics/FS/BTRFS"
	"github.com/aarsakian/FileSystemForensics/FS/NTFS/MFT"
)

type NTFSRecord struct {
	*MFT.Record
}

type BTRFSRecord struct {
	*fstree.FileDirEntry
}

func (btrfsRecord BTRFSRecord) FindAttribute(attrName string) Attribute {
	return btrfsRecord.FileDirEntry.FindAttribute(attrName)
}

func (btrfsRecord BTRFSRecord) GetLinkedRecords() []Record {
	var linkedRecords []Record
	for _, linkedRecord := range btrfsRecord.FileDirEntry.GetLinkedRecords() {
		linkedRecords = append(linkedRecords, BTRFSRecord{linkedRecord})
	}
	return linkedRecords
}

func (ntfsRecord NTFSRecord) GetLinkedRecords() []Record {
	var linkedRecords []Record
	for _, linkedRecord := range ntfsRecord.Record.LinkedRecords {
		linkedRecords = append(linkedRecords, NTFSRecord{linkedRecord})
	}
	return linkedRecords
}

func (ntfsRecord NTFSRecord) FindAttribute(attrName string) Attribute {
	return ntfsRecord.Record.FindAttribute(attrName)
}
