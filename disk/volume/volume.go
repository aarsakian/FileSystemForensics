package volume

import (
	metadata "github.com/aarsakian/FileSystemForensics/FS"
	"github.com/aarsakian/FileSystemForensics/readers"
)

type Volume interface {
	Process(readers.DiskReader, int64, []int, int, int)
	GetSectorsPerCluster() int
	GetBytesPerSector() uint64
	GetInfo() string
	GetFS() []metadata.Record
	GetFSOffset() int64
	GetUnallocatedClusters() []int
	GetSignature() string
	GetLogicalToPhysicalMap() map[uint64]metadata.Chunk
}
