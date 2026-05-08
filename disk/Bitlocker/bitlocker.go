package bitlocker

import (
	"errors"
	"fmt"

	datum "github.com/aarsakian/FileSystemForensics/disk/Bitlocker/datums"
	"github.com/aarsakian/FileSystemForensics/readers"
	"github.com/aarsakian/FileSystemForensics/utils"
)

// Based on libbde and dislocker  documentation

// All structs assume little-endian encoding and packed layout.
// Use encoding/binary with binary.LittleEndian.

// ============================================================================
// Volume Header - Windows 7 (512 bytes)
// ============================================================================

// VolumeHeaderWindows7 represents the BitLocker volume header for Windows 7.
// The header is 512 bytes total and is based on FAT32/NTFS BPB with BitLocker extensions.
type VolumeHeaderWindows7 struct {
	// Boot entry point and BIOS parameter block (BPB)
	BootEntryPoint      [3]byte // 0x00: "\xeb\x58\x90"
	FileSystemSignature [8]byte // 0x03: "-FVE-FS-"
	BytesPerSector      uint16  // 0x0B: Bytes per sector
	SectorsPerCluster   uint8   // 0x0D: Sectors per cluster
	ReservedSectors     uint16  // 0x0E: Reserved sectors (typically 0x00)
	FATs                uint8   // 0x10: Number of FATs (typically 0x00)
	RootDirEntries      uint16  // 0x11: Root directory entries (typically 0)
	TotalSectors16      uint16  // 0x13: Total sectors 16-bit (typically 0)
	MediaDescriptor     uint8   // 0x15: Media descriptor
	SectorsPerFAT       uint16  // 0x16: Sectors per FAT (typically 0x00)
	SectorsPerTrack     uint16  // 0x18: Sectors per track (typically 0x3f)
	NumberOfHeads       uint16  // 0x1A: Number of heads
	HiddenSectors       uint32  // 0x1C: Number of hidden sectors (volume start sector number)
	TotalSectors32      uint32  // 0x20: Total sectors 32-bit (typically 0x00)

	// FAT32 Extended BPB (unknown fields)
	SectorsPerFAT32  uint32   // 0x24: 0x1fe0 typically
	FATFlags         uint16   // 0x28: FAT Flags (only used during FAT12/16 conversion)
	Version          uint16   // 0x2A: Version (typically 0)
	RootDirCluster   uint32   // 0x2C: Cluster number of root directory start
	FSInfoSector     uint16   // 0x30: Sector number of FS Information Sector (typically 0x0001)
	BackupBootSector uint16   // 0x32: Sector number of backup boot sector (typically 0x0006)
	Reserved         [12]byte // 0x34: Unknown (Reserved)

	// Physical drive and signature information
	PhysicalDriveNumber uint8    // 0x40: Physical Drive Number (0x80)
	ReservedByte        uint8    // 0x41: Unknown (Reserved)
	ExtendedBootSig     uint8    // 0x42: Extended boot signature (0x29)
	VolumeSerialNumber  uint32   // 0x43: Volume serial number
	VolumeLabel         [11]byte // 0x47: Volume label ("NO NAME     ")
	FileSystemType      [8]byte  // 0x52: File system type ("FAT32   ")

	// Bootcode
	Bootcode [70]byte // 0x5A: Bootcode (offset 0x5A to 0xA3)

	// BitLocker specific fields
	BitLockerID        [16]byte  // 0xA0 (160): BitLocker identifier GUID
	FVEMetadataOffsets [3]uint64 // 0xB0 (176): FVE metadata block 1 offset (from volume start)

	// Additional bootcode
	AdditionalBootcode [307]byte // 0xC8 (200): Unknown (part of bootcode)
	UnknownBytes       [3]byte   // 0x1F7 (507): Unknown

	// Boot signature
	BootSignature [2]byte // 0x1FE (510): 0x55 0xaa
}

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
	MetadataSize       uint32   // 0x00: Total size of metadata including this header
	Version            uint32   // 0x04: Version (0x00000001 for Windows Vista/7)
	MetadataHeaderSize uint32   // 0x08: Size of metadata header (0x00000030 = 48 bytes)
	MetadataSizeCopy   uint32   // 0x0C: Copy of metadata size
	VolumeIdentifier   [16]byte // 0x10: Volume GUID identifier
	NextNonceCounter   uint32   // 0x20: Next nonce counter for AES-CCM
	EncryptionMethod   uint32   // 0x24: Encryption method (see EncryptionMethods)
	CreationTime       uint64   // 0x28: Creation time (Windows FILETIME)
}

// AESCCMKeyContainer is the unencrypted data inside AES-CCM.
type AESCCMKeyContainer struct {
	MAC              [16]byte // Message Authentication Code
	Size             uint32   // Size of container data (excluding MAC)
	Version          uint16   // Version (typically 1)
	Unknown          uint16   // Unknown field
	EncryptionMethod uint32   // Encryption method for the contained key
	UnencryptedData  []byte   // Key data (variable size)
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
	EncryptionMethod   uint32            // 0x24: Encryption method
	CreationTime       utils.WindowsTime // 0x28: Creation time (Windows FILETIME)
	// Additional fields may follow in Windows 8+ versions
}

// ============================================================================
// Windows Vista Volume Header (512 bytes) - For Reference
// ============================================================================

// VolumeHeaderVista represents the BitLocker volume header for Windows Vista.
// The header is 512 bytes total and is based on NTFS BPB.
// Note: Not actively parsed, included for documentation/version detection.
type VolumeHeaderVista struct {
	BootEntryPoint      [3]byte // 0x00: "\xeb\x52\x90"
	FileSystemSignature [8]byte // 0x03: "-FVE-FS-"
	BytesPerSector      uint16  // 0x0B: Bytes per sector
	SectorsPerCluster   uint8   // 0x0D: Sectors per cluster
	ReservedSectors     uint16  // 0x0E: Reserved sectors
	FATs                uint8   // 0x10: Number of FATs
	RootDirEntries      uint16  // 0x11: Root directory entries
	TotalSectors16      uint16  // 0x13: Total sectors 16-bit
	MediaDescriptor     uint8   // 0x15: Media descriptor
	SectorsPerFAT       uint16  // 0x16: Sectors per FAT
	SectorsPerTrack     uint16  // 0x18: Sectors per track
	NumberOfHeads       uint16  // 0x1A: Number of heads
	HiddenSectors       uint32  // 0x1C: Number of hidden sectors
	TotalSectors32      uint32  // 0x20: Total sectors 32-bit

	// NTFS BPB
	Unknown1            uint32    // 0x24: Unknown
	Unknown2            uint32    // 0x28: Unknown
	Unknown3            uint64    // 0x2C: Total sectors 64-bit
	MFTClusterNumber    uint64    // 0x34: MFT cluster number
	FVEMetadata1Cluster uint64    // 0x3C: FVE metadata block 1 cluster number
	MFTEntrySize        uint8     // 0x44: MFT entry size
	Unknown4            [3]byte   // 0x45: Unknown
	IndexEntrySize      uint8     // 0x48: Index entry size
	Unknown5            [3]byte   // 0x49: Unknown
	VolumeSerialNumber  uint64    // 0x4C: NTFS volume serial number
	Checksum            uint32    // 0x54: Checksum
	BootCode            [426]byte // 0x58: Bootcode
	BootSignature       [2]byte   // 0x1FE (510): 0x55 0xaa
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

// ============================================================================
// High-level Volume Structure
// ============================================================================

// Volume represents a BitLocker encrypted volume (supports Windows 7+).
// For Windows Vista, use alternate parsing logic.
type Volume struct {
	Header         VolumeHeaderWindows7 // Works for Windows 7+ (includes Windows 8/10)
	MetadataBlocks []MetadataBlock
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

// ============================================================================
// BitLocker External Key (BEK) File Format
// ============================================================================

// BEKFileHeaderV1 represents the header of a .BEK file (external/startup key).
// Size: 48 bytes
// Used for all Windows versions (Vista through Windows 10+)
type BEKFileHeaderV1 struct {
	MetadataSize       uint32   // Size of remaining data including this field
	Version            uint32   // Version (0x00000001)
	MetadataHeaderSize uint32   // Size of metadata header (0x00000030 = 48 bytes)
	MetadataSizeCopy   uint32   // Copy of metadata size
	VolumeIdentifier   [16]byte // Volume GUID identifier (must match FVE metadata)
	NextNonceCounter   uint32   // Next nonce counter for AES-CCM
	EncryptionMethod   uint32   // Encryption method
	CreationTime       uint64   // Creation time (Windows FILETIME)
}

type BitlockerValidations struct {
	Size    uint16
	Version uint16
	Crc32   uint32
}

// ============================================================================
// Version Detection and Support Notes
// ============================================================================

// BitLockerVersionInfo provides information about supported BitLocker versions
type BitLockerVersionInfo struct {
	// Windows Vista: Metadata block header version 1
	// - FVEMetadataBlockHeaderV1 with MFT mirror cluster information
	// - FVEMetadataHeaderV1 with version 0x00000001
	// - First 16 sectors stored unencrypted (virtual sectors)

	// Windows 7: Metadata block header version 2
	// - FVEMetadataBlockHeaderV2 with encrypted volume size and header sector count
	// - FVEMetadataHeaderV1 with version 0x00000001
	// - Encrypted volume header stored at offset from VolumeHeaderOffset
	// - FVE metadata block size: typically 65536 bytes (64 KiB)

	// Windows 8: Metadata block header version 2
	// - Same as Windows 7 with additional padding (0-byte values) in metadata blocks
	// - FVEMetadataHeaderV1 with version 0x00000001
	// - Support for new encryption methods and key protectors

	// Windows 10: Metadata block header version 2
	// - No FVE Volume header block (entry type 0x000F not present)
	// - Volume header sectors count in FVEMetadataBlockHeaderV2 determines size
	// - FVEMetadataHeaderV3 (version 0x00000003) may be used
	// - Additional improvements and compatibility with newer systems
}

// ============================================================================
// Utility Constants for Version Detection
// ============================================================================

const (
	// BitLocker GUID for standard Windows volumes
	BitLockerGUID = "4967d63b-2e29-4ad8-8399-f6a339e3d001"

	// BitLocker To Go GUID (removable media)
	BitLockerToGoGUID = "4967d63b-2e29-4ad8-8399-f6a339e3d01"

	// BitLocker Used Disk Space Only encryption GUID
	BitLockerUsedSpaceOnlyGUID = "92a84d3b-dd80-4d0e-9e4e-b1e3284eaed8"
)

// MetadataHeaderVersion returns the metadata header version from the raw bytes
func MetadataHeaderVersion(headerBytes []byte) uint16 {
	if len(headerBytes) < 12 {
		return 0
	}
	// Version is at offset 0x0A (10)
	return uint16(headerBytes[10]) | (uint16(headerBytes[11]) << 8)
}

// MetadataVersion returns the metadata (inner header) version from the raw bytes
func MetadataVersion(metadataBytes []byte) uint32 {
	if len(metadataBytes) < 8 {
		return 0
	}
	// Version is at offset 0x04 (4)
	return uint32(metadataBytes[4]) | (uint32(metadataBytes[5]) << 8) |
		(uint32(metadataBytes[6]) << 16) | (uint32(metadataBytes[7]) << 24)
}

func (volume *Volume) ProcessHeader(data []byte) {
	header := new(VolumeHeaderWindows7)
	utils.Unmarshal(data, header)
	volume.Header = *header
}

func (volume *Volume) Process(hD readers.DiskReader, volumeStartOffset int64) error {
	// Read the first metadata block header (at offset specified in volume header)
	var size int64
	for _, fveMetadataOffset := range volume.Header.FVEMetadataOffsets {
		if fveMetadataOffset != 0 {
			metadataBlockHeaderData, err := hD.ReadFile(int64(fveMetadataOffset)+volumeStartOffset, 112)
			if err != nil {
				return err
			}
			metadataBlockHeader := new(FVEMetadataBlockHeaderV2)
			metadataBlockHeader.Process(metadataBlockHeaderData)
			if metadataBlockHeader.Version == 2 {
				size = int64(metadataBlockHeader.Size << 4) // Size is in 16-sector units (512 bytes per sector)
			} else {
				size = int64(metadataBlockHeader.Size)
			}
			data, _ := hD.ReadFile(int64(fveMetadataOffset)+volumeStartOffset+size, 8)
			bv := new(BitlockerValidations)
			utils.Unmarshal(data, bv)

			data, _ = hD.ReadFile(int64(fveMetadataOffset)+volumeStartOffset, int(size))
			fmt.Println(size, utils.CalcCRC32IEEE(data))

			metadataBlock := MetadataBlock{Header: *metadataBlockHeader}
			if err := metadataBlock.ParseEntries(data); err != nil {
				return err
			}
			volume.MetadataBlocks = append(volume.MetadataBlocks, metadataBlock)
		}
	}

	/*for _, metadataBlock := range volume.MetadataBlocks {

		datasetData, _ := hD.ReadFile(int64(metadataBlock.Header.VolumeHeaderOffset)+volumeStartOffset, size)
		dataset := new(FVEMetadataHeaderV3)
		utils.Unmarshal(datasetData, dataset)


	}*/

	/*for _, datum := range volume.MetadataBlocks[0].Datums {
		fmt.Printf("%s\n", datum.GetInfo())
	}*/
	return nil
}

func (metadataBlockHeader *FVEMetadataBlockHeaderV2) Process(data []byte) {
	utils.Unmarshal(data, metadataBlockHeader)
	dataset := new(FVEMetadataHeaderV3)
	utils.Unmarshal(data[64:], dataset)
	metadataBlockHeader.DataSets = dataset
	dataset.ShowInfo()

}

func (dataset FVEMetadataHeaderV3) ShowInfo() {
	fmt.Printf("Windows FILETIME: %s GUID %s\n",
		dataset.CreationTime.ConvertToIsoTime(), utils.StringifyGUID(dataset.VolumeIdentifier[:]))
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
		fmt.Printf("%s\n", datum.GetInfo())
		metadataBlock.Datums = append(metadataBlock.Datums, datum)
		offset += int(datumHeader.EntrySize)
	}

	return nil
}

/*
case EncryptionMethodStretchKey1:
		return "Stretch key variant 1"
	case EncryptionMethodStretchKey2:
		return "Stretch key variant 2"
	case EncryptionMethodAESCCM1:
		return "AES-CCM 256-bit"
	case EncryptionMethodAESCCM2:
		return "AES-CCM 256-bit variant 1"
	case EncryptionMethodAESCCM3:
		return "AES-CCM 256-bit variant 2"
	case EncryptionMethodAESCCM4:
		return "AES-CCM 256-bit variant 3"
	case EncryptionMethodAESCCM5:
		return "AES-CCM 256-bit variant 4"
	case EncryptionMethodAESCCM6:
		return "AES-CCM 256-bit variant 5"
	case EncryptionMethodAESCBCElephant128:
		return "AES-CBC 128-bit with Elephant Diffuser"
	case EncryptionMethodAESCBCElephant256:
		return "AES-CBC 256-bit with Elephant Diffuser"
	case EncryptionMethodAESCBC128:
		return "AES-CBC 128-bit"
	case EncryptionMethodAESCBC256:
		return "AES-CBC 256-bit"
	case EncryptionMethodAESXTS128:
		return "AES-XTS 128-bit"
	case EncryptionMethodAESXTS256:
		return "AES-XTS 256-bit"*/
