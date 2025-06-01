package volume

import (
	"github.com/aarsakian/FileSystemForensics/FS/NTFS/MFT"
	"github.com/aarsakian/FileSystemForensics/img"
)

type Volume interface {
	Process(img.DiskReader, int64, []int, int, int)
	GetSectorsPerCluster() int
	GetBytesPerSector() uint64
	GetInfo() string
	GetFS() []MFT.Record
	CollectUnallocated(img.DiskReader, int64, chan<- []byte)
	GetSignature() string
}
