package volume

import (
	metadata "github.com/aarsakian/FileSystemForensics/FS"
	"github.com/aarsakian/FileSystemForensics/img"
)

type Volume interface {
	Process(img.DiskReader, int64, []int, int, int)
	GetSectorsPerCluster() int
	GetBytesPerSector() uint64
	GetInfo() string
	GetFS() []metadata.Record
	GetUnallocatedClusters() []int
	GetSignature() string
	GetLogicalToPhysicalMap() map[uint64]metadata.Chunk
}
