package bitlocker

import (
	"errors"
	"fmt"

	"github.com/aarsakian/FileSystemForensics/disk/Bitlocker/datums"
	datum "github.com/aarsakian/FileSystemForensics/disk/Bitlocker/datums"
	"github.com/aarsakian/FileSystemForensics/utils"
)

// ============================================================================
// FVE Metadata Block Header - Version 2 (Windows 7) - 64 bytes
// ============================================================================

// FVEMetadataBlockHeaderV2 represents the FVE metadata block header for Windows 7.
// Each BitLocker volume has 3 copies of this structure for redundancy.
// equal to _bitlocker_information in dislocker
type FVEMetadataBlockHeaderV2 struct {
	Signature             [8]byte   // 0x00: "-FVE-FS-"
	Size                  uint16    // 0x08: Size of block header
	Version               uint16    // 0x0A: Version (0x0002 for Windows 7)
	ProtectionStatus      uint16    // 0x0C: Unknown - 0x04 typically, 0x05 in partial decrypted volume
	ProtectionStatusCopy  uint16    // 0x0E: Unknown copy - 0x04 typically, 0x01 in partial decrypted volume
	EncryptedVolumeSize   uint64    // 0x10: Encrypted volume size in bytes (bytes still encrypted/to be decrypted)
	ConvertSize           uint32    // 0x18: Unknown (reserved)
	NumberOfHeaderSectors uint32    // 0x1C: Number of volume header sectors (typically 8192/512 = 16)
	FVEMetadataOffsets    [3]uint64 // 0x20: FVE metadata block 1 offset (from volume start)
	VolumeHeaderOffset    uint64    // 0x38: Volume header offset (encrypted first sectors location)
	DataSets              *FVEMetadataHeaderV3
}

// ============================================================================
// FVE Metadata Header - Version 1 (Windows 7) - 48 bytes
// ============================================================================

// FVEMetadataHeaderV1 contains metadata for the volume such as GUID and encryption method.
// dataset
type FVEMetadataHeaderV1 struct {
	MetadataSize       uint32            // 0x00: Total size of metadata including this header
	Version            uint32            // 0x04: Version (0x00000001 for Windows Vista/7)
	MetadataHeaderSize uint32            // 0x08: Size of metadata header (0x00000030 = 48 bytes)
	MetadataSizeCopy   uint32            // 0x0C: Copy of metadata size
	VolumeIdentifier   [16]byte          // 0x10: Volume GUID identifier
	NextNonceCounter   uint32            // 0x20: Next nonce counter for AES-CCM
	EncryptionMethod   [4]byte           // 0x24: Encryption method (see EncryptionMethods)
	CreationTime       utils.WindowsTime // 0x28: Creation time (Windows FILETIME)
}

// ============================================================================
// FVE Metadata Header - Version 3 (Windows 8+) - 48+ bytes
// ============================================================================

// FVEMetadataHeaderV3 contains metadata for Windows 8 and later versions.
// This version extends V1 with additional fields and support for newer encryption methods.
// Note: Version 3 was documented in February 2022 update to libbde specification.
type FVEMetadataHeaderV3 struct {
	MetadataSize       uint32            // 0x00: Total size of metadata including this header
	Version            uint32            // 0x04: Version (0x00000003 for Windows 8+)
	MetadataHeaderSize uint32            // 0x08: Size of metadata header
	MetadataSizeCopy   uint32            // 0x0C: Copy of metadata size
	VolumeIdentifier   [16]byte          // 0x10: Volume GUID identifier
	NextNonceCounter   uint32            // 0x20: Next nonce counter for AES-CCM
	EncryptionMethod   [4]byte           // 0x24: Encryption method
	CreationTime       utils.WindowsTime // 0x28: Creation time (Windows FILETIME)
	// Additional fields may follow in Windows 8+ versions
}

// ============================================================================
// FVE Metadata Block Header - Version 1 (Windows Vista) - 64 bytes
// ============================================================================

// FVEMetadataBlockHeaderV1 represents the FVE metadata block header for Windows Vista.
// Each BitLocker volume has 3 copies of this structure for redundancy.
// 0x40 bitlocker_dataset
type FVEMetadataBlockHeaderV1 struct {
	Signature          [8]byte  // 0x00: "-FVE-FS-"
	Size               uint16   // 0x08: Size of block header
	Version            uint16   // 0x0A: Version (0x0001 for Windows Vista)
	Unknown1           uint16   // 0x0C: Unknown - 0x04 typically
	Unknown2           uint16   // 0x0E: Unknown - 0x04 typically
	UnknownReserved    [16]byte // 0x10: Unknown (empty values)
	FVEMetadataOffset1 uint64   // 0x20: FVE metadata block 1 offset
	FVEMetadataOffset2 uint64   // 0x28: FVE metadata block 2 offset
	FVEMetadataOffset3 uint64   // 0x30: FVE metadata block 3 offset
	MFTMirrorCluster   uint64   // 0x38: MFT mirror cluster block number
}

// MetadataBlock represents one of the three FVE metadata blocks.
// The metadata header version determines how to parse the block.
type MetadataBlock struct {
	Header       FVEMetadataBlockHeaderV2 // Works for Windows 7+ (V2 = Windows 7, V2 for Windows 8/10)
	MetadataV1   *FVEMetadataHeaderV1     // For Windows 7 and early Windows 8
	MetadataV3   *FVEMetadataHeaderV3     // For Windows 8+ (if version 3 is detected)
	Datums       []datum.Datum            //entries of datums
	PaddingBytes []byte                   // Padding (seen in Windows 8 metadata blocks)
}

// ====
func (metadataBlockHeader *FVEMetadataBlockHeaderV2) Process(data []byte) {
	utils.Unmarshal(data, metadataBlockHeader)

}

func (dataset FVEMetadataHeaderV1) GetInfo() {
	fmt.Printf("V1 Creation Time: %s GUID %s Encryption Method: %s\n",
		dataset.CreationTime.ConvertToIsoTime(), utils.StringifyGUID(dataset.VolumeIdentifier[:]),
		datums.GetEncryptionMethod(dataset.EncryptionMethod[:]))
}

func (dataset FVEMetadataHeaderV3) GetInfo() {
	fmt.Printf("V3 Creation Time: %s GUID %s Encryption Method: %s\n",
		dataset.CreationTime.ConvertToIsoTime(), utils.StringifyGUID(dataset.VolumeIdentifier[:]),
		datums.GetEncryptionMethod(dataset.EncryptionMethod[:]))
}

func (metadataBlock *MetadataBlock) ParseEntries(raw []byte) error {
	if len(raw) < 64 {
		return errors.New("metadata block data too small")
	}

	offset := 64
	if len(raw) <= offset {
		return nil
	}

	metadataVersion := MetadataVersion(raw[offset:])
	switch metadataVersion {
	case 1:
		metadataBlock.MetadataV1 = new(FVEMetadataHeaderV1)
		if _, err := utils.Unmarshal(raw[offset:], metadataBlock.MetadataV1); err != nil {
			return err
		}
		offset += int(metadataBlock.MetadataV1.MetadataHeaderSize)
	case 3:
		metadataBlock.MetadataV3 = new(FVEMetadataHeaderV3)
		if _, err := utils.Unmarshal(raw[offset:], metadataBlock.MetadataV3); err != nil {
			return err
		}
		offset += int(metadataBlock.MetadataV3.MetadataHeaderSize)
	default:
		fallback := new(FVEMetadataHeaderV1)
		if _, err := utils.Unmarshal(raw[offset:], fallback); err == nil && fallback.MetadataHeaderSize >= 48 {
			metadataBlock.MetadataV1 = fallback
			offset += int(fallback.MetadataHeaderSize)
		}
	}

	for offset < len(raw) {
		datumHeader := new(datum.DatumHeader)
		err := datumHeader.Process(raw[offset:])
		if err != nil {
			break
		}
		if datumHeader.EntrySize == 0 {
			break
		}
		datum, err := datum.CreateDatum(*datumHeader, raw[offset+8:offset+int(datumHeader.EntrySize)])
		if err != nil {
			break
		}

		metadataBlock.Datums = append(metadataBlock.Datums, datum)
		offset += int(datumHeader.EntrySize)
	}

	return nil
}

func (metadataBlock MetadataBlock) GetInfo() {
	if metadataBlock.MetadataV1 != nil {
		metadataBlock.MetadataV1.GetInfo()

	} else if metadataBlock.MetadataV3 != nil {
		metadataBlock.MetadataV3.GetInfo()
	}
	for _, datum := range metadataBlock.Datums {
		fmt.Printf("Datum Info: %s\n", datum.GetInfo())
	}
}
