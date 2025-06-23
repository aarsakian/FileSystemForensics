package metadata

import "github.com/aarsakian/FileSystemForensics/FS/NTFS/MFT"

type NTFSRecord struct {
	*MFT.Record
}

func (ntfsRecord NTFSRecord) GetLinkedRecords() []Record {
	var linkedRecords []Record
	for _, linkedRecord := range ntfsRecord.LinkedRecords {
		linkedRecords = append(linkedRecords, NTFSRecord{linkedRecord})
	}
	return linkedRecords
}

func (ntfsRecord NTFSRecord) FindAttribute(attrName string) Attribute {
	return ntfsRecord.Record.FindAttribute(attrName)
}
