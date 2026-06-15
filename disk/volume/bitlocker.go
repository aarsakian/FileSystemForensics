package volume

import (
	"errors"
	"fmt"

	metadata "github.com/aarsakian/FileSystemForensics/FS"
	bitlockerPkg "github.com/aarsakian/FileSystemForensics/disk/Bitlocker"
	"github.com/aarsakian/FileSystemForensics/logger"
	"github.com/aarsakian/FileSystemForensics/readers"
	"github.com/aarsakian/FileSystemForensics/utils"
)

// Bitlocker is a wrapper volume that represents a BitLocker-encrypted volume
// and (after unlocking) delegates calls to the underlying filesystem volume.
type Bitlocker struct {
	BL             *bitlockerPkg.Volume
	Inner          Volume
	partitionStart int64
}

func (b *Bitlocker) Process(hD readers.DiskReader, partitionOffsetB int64, MFTSelectedEntries []int,
	fromMFTEntry int, toMFTEntry int) error {
	// Parse BitLocker metadata blocks
	if b.BL == nil {
		b.BL = new(bitlockerPkg.Volume)
	}
	if err := b.BL.Process(hD, partitionOffsetB); err != nil {
		return err
	}
	b.partitionStart = partitionOffsetB
	logger.FSLogger.Info(fmt.Sprintf("BitLocker volume discovered at %d", partitionOffsetB))
	// Underlying FS processing happens after decrypt/unlock (Decrypt method)
	return nil
}

func (b *Bitlocker) ProcessHeader(data []byte) {
	if b.BL == nil {
		b.BL = new(bitlockerPkg.Volume)
	}
	b.BL.ProcessHeader(data)
}

func (b *Bitlocker) GetSectorsPerCluster() int {
	if b.Inner != nil {
		return b.Inner.GetSectorsPerCluster()
	}
	if b.BL != nil {
		return int(b.BL.Header.SectorsPerCluster)
	}
	return 0
}

func (b *Bitlocker) GetBytesPerSector() uint64 {
	if b.Inner != nil {
		return b.Inner.GetBytesPerSector()
	}
	if b.BL != nil {
		return uint64(b.BL.Header.BytesPerSector)
	}
	return 512
}

func (b *Bitlocker) GetInfo() string {
	if b.BL == nil {
		return "BitLocker (no metadata)"
	}
	return fmt.Sprintf("BitLocker GUID %s", utils.StringifyGUID(b.BL.Header.BitLockerID[:]))
}

func (b *Bitlocker) GetFS() []metadata.Record {
	if b.Inner != nil {
		return b.Inner.GetFS()
	}
	return []metadata.Record{}
}

func (b *Bitlocker) GetFSOffset() int64 {
	if b.Inner != nil {
		return b.Inner.GetFSOffset()
	}
	return 0
}

func (b *Bitlocker) GetUnallocatedClusters() []int {
	if b.Inner != nil {
		return b.Inner.GetUnallocatedClusters()
	}
	return []int{}
}

func (b *Bitlocker) GetSignature() string {
	return "BitLocker"
}

func (b *Bitlocker) GetLogicalToPhysicalMap() map[uint64]metadata.Chunk {
	if b.Inner != nil {
		return b.Inner.GetLogicalToPhysicalMap()
	}
	return map[uint64]metadata.Chunk{}
}

// Decrypt attempts to unlock the BitLocker volume using provided credentials.
// NOTE: This scaffolding calls into the bitlocker package to obtain the FVEK
// but does not yet implement the transparent decryption layer that would
// instantiate the underlying filesystem volume. That integration is TODO.
func (b *Bitlocker) Decrypt(password string, recoverykey string) error {
	if b.BL == nil {
		return errors.New("no BitLocker metadata parsed")
	}
	if err := b.BL.Decrypt(password, recoverykey); err != nil {
		return err
	}
	// At this point b.BL.Key contains the decrypted FVEK. To continue, a
	// transparent decrypting reader must be created that wraps the original
	// readers.DiskReader and decrypts read blocks using the FVEK. Implementing
	// that layer is non-trivial and is left for a follow-up change.
	logger.FSLogger.Info("BitLocker unlocked (FVEK available) - instantiate underlying FS next")
	return nil
}
