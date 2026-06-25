package filters

import (
	metadata "github.com/aarsakian/FileSystemForensics/FS"
	"github.com/aarsakian/FileSystemForensics/disk"
	"github.com/aarsakian/FileSystemForensics/signatures"
)

type Filter interface {
	Execute(records []metadata.Record) []metadata.Record
}

type NameFilter struct {
	Filenames []string
}

type SignatureFilter struct {
	Sgm         signatures.SignatureManager
	Disk        disk.Disk
	PartitionId int
	Level       string
}

func (signatureFilter SignatureFilter) Execute(records []metadata.Record) []metadata.Record {
	physicalToLogicalMap, _ := signatureFilter.Disk.GetLogicalToPhysicalMap(signatureFilter.PartitionId)
	partition := signatureFilter.Disk.Partitions[signatureFilter.PartitionId]

	clusterSizeB := int(partition.GetVolume().GetBytesPerSector() * uint64(partition.GetVolume().GetSectorsPerCluster()))
	partitionOffset := int64(partition.GetOffset() * partition.GetVolume().GetBytesPerSector())

	return metadata.FilterBySignatures(records, signatureFilter.Sgm, signatureFilter.Level,
		clusterSizeB, partitionOffset, physicalToLogicalMap, signatureFilter.Disk.Handler)
}

func (nameFilter NameFilter) Execute(records []metadata.Record) []metadata.Record {
	return metadata.FilterByNames(records, nameFilter.Filenames)
}

type PathFilter struct {
	NamePath string
}

func (pathFilter PathFilter) Execute(records []metadata.Record) []metadata.Record {
	return metadata.FilterByPath(records, pathFilter.NamePath)
}

type ExtensionsFilter struct {
	Extensions []string
}

func (extensionsFilter ExtensionsFilter) Execute(records []metadata.Record) []metadata.Record {
	return metadata.FilterByExtensions(records, extensionsFilter.Extensions)
}

type OrphansFilter struct {
	Include bool
}

func (orphansFilter OrphansFilter) Execute(records []metadata.Record) []metadata.Record {
	if orphansFilter.Include {
		return metadata.FilterOrphans(records)
	}
	return records
}

type DeletedFilter struct {
	Include bool
}

func (deletedFilter DeletedFilter) Execute(records []metadata.Record) []metadata.Record {
	if deletedFilter.Include {
		return metadata.FilterDeleted(records, deletedFilter.Include)
	}
	return records
}

type FoldersFilter struct {
	Include bool
}

func (foldersFilter FoldersFilter) Execute(records []metadata.Record) []metadata.Record {
	if !foldersFilter.Include {
		return metadata.FilterOutFolders(records)
	}
	return records
}

type PrefixesSuffixesFilter struct {
	Prefixes []string
	Suffixes []string
}

func (prefSufFilter PrefixesSuffixesFilter) Execute(records []metadata.Record) []metadata.Record {
	for idx, prefix := range prefSufFilter.Prefixes {
		records = metadata.FilterByPrefixSuffix(records, prefix, prefSufFilter.Suffixes[idx])
	}

	return records

}

type EntriesFilter struct {
	Entries []int
}

func (ef EntriesFilter) Execute(records []metadata.Record) []metadata.Record {
	for _, entry := range ef.Entries {
		records = metadata.FilterByEntries(records, entry)
	}
	return records
}
