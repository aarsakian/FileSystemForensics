package metadata

import (
	"github.com/aarsakian/FileSystemForensics/FS/NTFS/MFT"
	"github.com/aarsakian/FileSystemForensics/img"
	"github.com/aarsakian/FileSystemForensics/utils"
)

type Attribute interface {
}

type Record interface {
	HasFilenameExtension(string) bool
	HasFilenames([]string) bool
	HasFilename(string) bool
	HasPath(string) bool
	HasParent() bool
	HasSuffix(string) bool
	HasPrefix(string) bool
	IsDeleted() bool
	IsFolder() bool
	GetFname() string
	GetID() int
	LocateData(img.DiskReader, int64, int, int, chan<- utils.AskedFile)
	LocateDataAsync(img.DiskReader, int64, int, chan<- []byte)
	GetLinkedRecords() []*MFT.Record
	GetLogicalFileSize() int64
	GetSequence() int
	FindAttribute(string) MFT.Attribute
	ShowAttributes(string)
	ShowTimestamps()
	ShowIsResident()
	ShowRunList()
	ShowFileSize()
	ShowVCNs()
	ShowIndex()
	ShowParentRecordInfo()
	ShowPath(int)
}

func FilterByExtensions(records []Record, extensions []string) []Record {
	var filteredRecords []Record
	for _, extension := range extensions {
		filteredRecords = append(filteredRecords, FilterByExtension(records, extension)...)
	}
	return filteredRecords
}

func FilterByExtension(records []Record, extension string) []Record {

	return utils.Filter(records, func(record Record) bool {
		return record.HasFilenameExtension(extension)
	})

}

func FilterByNames(records []Record, filenames []string) []Record {

	return utils.Filter(records, func(record Record) bool {
		return record.HasFilenames(filenames)
	})

}

func FilterByPath(records []Record, filespath string) []Record {
	return utils.Filter(records, func(record Record) bool {
		return record.HasPath(filespath)
	})
}

func FilterByName(records []Record, filename string) []Record {
	return utils.Filter(records, func(record Record) bool {
		return record.HasFilename(filename)
	})

}

func FilterOrphans(records []Record) []Record {
	return utils.Filter(records, func(record Record) bool {
		return record.IsDeleted() && record.HasParent()
	})
}

func FilterByPrefixSuffix(records []Record, prefix string, suffix string) []Record {

	return utils.Filter(records, func(record Record) bool {
		return record.HasPrefix(prefix) && record.HasSuffix(suffix)
	})

}

func FilterOutFiles(records []Record) []Record {
	return utils.Filter(records, func(record Record) bool {
		return record.IsFolder()
	})
}

func FilterOutFolders(records []Record) []Record {
	return utils.Filter(records, func(record Record) bool {
		return !record.IsFolder()
	})
}

func FilterDeleted(records []Record, includeDeleted bool) []Record {
	return utils.Filter(records, func(record Record) bool {
		if includeDeleted {
			return record.IsDeleted()
		} else {
			return !record.IsDeleted()
		}

	})
}
