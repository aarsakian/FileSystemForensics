package volume

import (
	"errors"
	"fmt"
	"math"

	metadata "github.com/aarsakian/FileSystemForensics/FS"
	bitlockerPkg "github.com/aarsakian/FileSystemForensics/disk/Bitlocker"
	"github.com/aarsakian/FileSystemForensics/logger"
	"github.com/aarsakian/FileSystemForensics/readers"
	"github.com/aarsakian/FileSystemForensics/utils"
)

// Bitlocker is a wrapper volume that represents a BitLocker-encrypted volume
// and (after unlocking) delegates calls to the underlying filesystem volume.
type Bitlocker struct {
	BL              *bitlockerPkg.Volume
	Inner           Volume
	partitionStartB int64
	handler         readers.DiskReader
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
	b.partitionStartB = partitionOffsetB
	b.handler = hD
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

func (b *Bitlocker) ReadFile(offset int64, size int) ([]byte, error) {
	if b.handler == nil {
		return nil, errors.New("no reader available for BitLocker volume")
	}
	return b.handler.ReadFile(offset, size)
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
		fmt.Printf("Failed to decrypt BitLocker volume: %v\n", err)
		return err
	}

	if b.handler == nil {
		return errors.New("no disk reader available for BitLocker decryption")
	}

	sectorSize := int64(b.BL.Header.BytesPerSector)
	if sectorSize == 0 {
		sectorSize = 512
	}

	decryptingReader, err := readers.NewDecryptingReader(
		b.handler, //original handler
		b.BL.Key,
		sectorSize,
		b.partitionStartB,
		b.BL.GetEncryptionMethod(),
	)
	if err != nil {
		return err
	}

	// Replace the handler with the decrypting reader so subsequent reads
	// on the BitLocker volume return decrypted data.
	b.handler = decryptingReader

	// If the underlying volume has already been detected, create it here.
	nativeVol, err := b.detectInnerVolume()
	if err != nil {
		return err
	}
	b.Inner = nativeVol

	// If the inner volume is an NTFS or BTRFS volume, process it now so the
	// filesystem metadata can be populated through the BitLocker wrapper.
	if b.Inner != nil {
		if err := b.Inner.Process(b.handler, b.partitionStartB, []int{}, 0, math.MaxUint32); err != nil {
			return err
		}
	}

	msg := "BitLocker unlocked and decrypting reader instantiated"

	logger.FSLogger.Info(msg)
	fmt.Printf("%s\n", msg)
	return nil
}

func (b *Bitlocker) detectInnerVolume() (Volume, error) {
	partitionOffsetB := b.partitionStartB
	volumeOffsetB := b.BL.GetVolumeOffset()
	if volumeOffsetB > 0 {
		partitionOffsetB += volumeOffsetB
	}
	data, err := b.handler.ReadFile(partitionOffsetB, 512)
	if err != nil {
		return nil, err
	}

	if len(data) >= 11 && string(data[3:11]) == "-FVE-FS-" {
		return nil, errors.New("nested BitLocker volumes are not supported")
	}

	if len(data) >= 11 && string(data[3:7]) == "NTFS" {
		ntfs := new(NTFS)
		ntfs.ProcessHeader(data)
		return ntfs, nil
	}

	btrfs := new(BTRFS)
	if len(data) >= 11 {
		if err := btrfs.ParseSuperblock(data); err == nil && btrfs.HasValidSignature() {
			return btrfs, nil
		}
	}

	return nil, errors.New("unsupported or unknown decrypted filesystem")
}

func (b Bitlocker) GetHandler() readers.DiskReader {
	return b.handler
}
