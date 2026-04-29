package bitlocker

import (
	"github.com/aarsakian/FileSystemForensics/readers"
	"github.com/aarsakian/FileSystemForensics/utils"
)

// BitLocker Windows 7 format structures
// Based on libbde documentation: https://github.com/libyal/libbde/blob/main/documentation/BitLocker%20Drive%20Encryption%20(BDE)%20format.asciidoc
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
	BitLockerID        [16]byte // 0xA0 (160): BitLocker identifier GUID
	FVEMetadataOffset1 uint64   // 0xB0 (176): FVE metadata block 1 offset (from volume start)
	FVEMetadataOffset2 uint64   // 0xB8 (184): FVE metadata block 2 offset (from volume start)
	FVEMetadataOffset3 uint64   // 0xC0 (192): FVE metadata block 3 offset (from volume start)

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
type FVEMetadataBlockHeaderV2 struct {
	Signature             [8]byte // 0x00: "-FVE-FS-"
	Size                  uint16  // 0x08: Size of block header
	Version               uint16  // 0x0A: Version (0x0002 for Windows 7)
	ProtectionStatus      uint16  // 0x0C: Unknown - 0x04 typically, 0x05 in partial decrypted volume
	ProtectionStatusCopy  uint16  // 0x0E: Unknown copy - 0x04 typically, 0x01 in partial decrypted volume
	EncryptedVolumeSize   uint64  // 0x10: Encrypted volume size in bytes (bytes still encrypted/to be decrypted)
	Unknown1              uint32  // 0x18: Unknown (reserved)
	NumberOfHeaderSectors uint32  // 0x1C: Number of volume header sectors (typically 8192/512 = 16)
	FVEMetadataOffset1    uint64  // 0x20: FVE metadata block 1 offset (from volume start)
	FVEMetadataOffset2    uint64  // 0x28: FVE metadata block 2 offset (from volume start)
	FVEMetadataOffset3    uint64  // 0x30: FVE metadata block 3 offset (from volume start)
	VolumeHeaderOffset    uint64  // 0x38: Volume header offset (encrypted first sectors location)
}

// ============================================================================
// FVE Metadata Header - Version 1 (Windows 7) - 48 bytes
// ============================================================================

// FVEMetadataHeaderV1 contains metadata for the volume such as GUID and encryption method.
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

// ============================================================================
// FVE Metadata Entry - Variable size
// ============================================================================

// FVEMetadataEntry represents a single metadata entry.
// Entries are variable-sized and contain different types of data.
type FVEMetadataEntry struct {
	EntrySize uint16 // 0x00: Size including this field
	EntryType uint16 // 0x02: Entry type (see MetadataEntryType)
	ValueType uint16 // 0x04: Value type (see MetadataValueType)
	Version   uint16 // 0x06: Version (typically 1, sometimes 3 for clear key VMK)
	Data      []byte // 0x08+: Variable-length data based on EntryType and ValueType
}

// ============================================================================
// Metadata Entry Type Constants
// ============================================================================

const (
	MetadataEntryTypeNone         uint16 = 0x0000 // Property entry
	MetadataEntryTypeVMK          uint16 = 0x0002 // Volume Master Key
	MetadataEntryTypeFVEK         uint16 = 0x0003 // Full Volume Encryption Key
	MetadataEntryTypeValidation   uint16 = 0x0004 // Validation data
	MetadataEntryTypeStartupKey   uint16 = 0x0006 // Startup key (external key)
	MetadataEntryTypeDescription  uint16 = 0x0007 // Description (drive label)
	MetadataEntryTypeFVEKBackup   uint16 = 0x000B // Backup FVEK
	MetadataEntryTypeVolumeHeader uint16 = 0x000F // Volume header block
)

// ============================================================================
// Metadata Value Type Constants
// ============================================================================

const (
	MetadataValueTypeErased        uint16 = 0x0000 // Erased entry
	MetadataValueTypeKey           uint16 = 0x0001 // Encryption key
	MetadataValueTypeUnicodeString uint16 = 0x0002 // Unicode string (UTF-16 LE with null terminator)
	MetadataValueTypeStretchKey    uint16 = 0x0003 // Stretch key (salt + AES-CCM encrypted key)
	MetadataValueTypeUseKey        uint16 = 0x0004 // Use key
	MetadataValueTypeAESCCMKey     uint16 = 0x0005 // AES-CCM encrypted key
	MetadataValueTypeTPMEncodedKey uint16 = 0x0006 // TPM encoded key
	MetadataValueTypeValidation    uint16 = 0x0007 // Validation
	MetadataValueTypeVMK           uint16 = 0x0008 // Volume Master Key structure
	MetadataValueTypeExternalKey   uint16 = 0x0009 // External key
	MetadataValueTypeUpdate        uint16 = 0x000A // Update entry
	MetadataValueTypeError         uint16 = 0x000B // Error entry
	MetadataValueTypeOffsetSize    uint16 = 0x000F // Offset and size (two 64-bit values)
)

// ============================================================================
// Encryption Method Constants
// ============================================================================

const (
	EncryptionMethodUnknown           uint32 = 0x0000 // Unknown or not encrypted
	EncryptionMethodStretchKey1       uint32 = 0x1000 // Stretch key variant 1
	EncryptionMethodStretchKey2       uint32 = 0x1001 // Stretch key variant 2
	EncryptionMethodAESCCM1           uint32 = 0x2000 // AES-CCM 256-bit
	EncryptionMethodAESCCM2           uint32 = 0x2001 // AES-CCM 256-bit variant 1
	EncryptionMethodAESCCM3           uint32 = 0x2002 // AES-CCM 256-bit variant 2
	EncryptionMethodAESCCM4           uint32 = 0x2003 // AES-CCM 256-bit variant 3
	EncryptionMethodAESCCM5           uint32 = 0x2004 // AES-CCM 256-bit variant 4
	EncryptionMethodAESCCM6           uint32 = 0x2005 // AES-CCM 256-bit variant 5
	EncryptionMethodAESCBCElephant128 uint32 = 0x8000 // AES-CBC 128-bit with Elephant Diffuser
	EncryptionMethodAESCBCElephant256 uint32 = 0x8001 // AES-CBC 256-bit with Elephant Diffuser
	EncryptionMethodAESCBC128         uint32 = 0x8002 // AES-CBC 128-bit
	EncryptionMethodAESCBC256         uint32 = 0x8003 // AES-CBC 256-bit
	EncryptionMethodAESXTS128         uint32 = 0x8004 // AES-XTS 128-bit
	EncryptionMethodAESXTS256         uint32 = 0x8005 // AES-XTS 256-bit
)

// ============================================================================
// Volume Master Key (VMK) Protection Type Constants
// ============================================================================

const (
	VMKProtectionTypeClearKey   uint16 = 0x0000 // Clear key (unprotected VMK)
	VMKProtectionTypeTPM        uint16 = 0x0100 // TPM-protected
	VMKProtectionTypeStartupKey uint16 = 0x0200 // Startup key-protected (USB)
	VMKProtectionTypeTPMPIN     uint16 = 0x0500 // TPM + PIN-protected
	VMKProtectionTypeRecoveryPW uint16 = 0x0800 // Recovery password-protected
	VMKProtectionTypePassword   uint16 = 0x2000 // User password-protected
)

// ============================================================================
// FVE Key Structure
// ============================================================================

// FVEKey represents a key with encryption method and key data.
type FVEKey struct {
	EncryptionMethod uint32 // Encryption method used
	KeyData          []byte // Variable-length key data
}

// ============================================================================
// FVE Stretch Key
// ============================================================================

// FVEStretchKey is used for password/recovery key protection.
// Value type: 0x0003
type FVEStretchKey struct {
	EncryptionMethod uint32   // Encryption method
	Salt             [16]byte // 16-byte salt
	EncryptedKey     []byte   // AES-CCM encrypted key follows (variable size)
}

// ============================================================================
// FVE AES-CCM Encrypted Key
// ============================================================================

// FVEAESCCMKey represents an AES-CCM encrypted key entry.
// Value type: 0x0005
type FVEAESCCMKey struct {
	NonceDateTime uint64 // FILETIME when nonce was created
	NonceCounter  uint32 // Nonce counter
	EncryptedData []byte // AES-CCM encrypted data (variable size)
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
// FVE Volume Master Key (VMK)
// ============================================================================

// FVEVolumeMasterKey represents the VMK entry.
// Value type: 0x0008
type FVEVolumeMasterKey struct {
	KeyIdentifier        [16]byte // GUID identifying this VMK
	LastModificationTime uint64   // FILETIME of last modification
	Unknown              uint16   // Unknown field
	ProtectionType       uint16   // Protection type (see VMKProtectionType constants)
	Properties           []byte   // Variable array of property entries (entry type 0x0000)
}

// ============================================================================
// FVE External Key (Startup Key)
// ============================================================================

// FVEExternalKey represents a startup key (USB key) entry.
// Value type: 0x0009
type FVEExternalKey struct {
	KeyIdentifier        [16]byte // GUID identifying this key
	LastModificationTime uint64   // FILETIME of last modification
	Properties           []byte   // Variable array of property entries
}

// ============================================================================
// FVE Volume Header Block
// ============================================================================

// FVEVolumeHeaderBlock specifies the location of the encrypted volume header.
// Value type: 0x000F
type FVEVolumeHeaderBlock struct {
	BlockOffset    uint64 // Offset where encrypted volume header is stored
	BlockSize      uint64 // Size of encrypted volume header in bytes
	Unknown1       uint16 // Unknown (entry count?)
	Unknown2       uint16 // Unknown (size of additional data?)
	AdditionalData []byte // Variable additional data (array of 14-byte entries)
}

// ============================================================================
// FVE Metadata Header - Version 3 (Windows 8+) - 48+ bytes
// ============================================================================

// FVEMetadataHeaderV3 contains metadata for Windows 8 and later versions.
// This version extends V1 with additional fields and support for newer encryption methods.
// Note: Version 3 was documented in February 2022 update to libbde specification.
type FVEMetadataHeaderV3 struct {
	MetadataSize       uint32   // 0x00: Total size of metadata including this header
	Version            uint32   // 0x04: Version (0x00000003 for Windows 8+)
	MetadataHeaderSize uint32   // 0x08: Size of metadata header
	MetadataSizeCopy   uint32   // 0x0C: Copy of metadata size
	VolumeIdentifier   [16]byte // 0x10: Volume GUID identifier
	NextNonceCounter   uint32   // 0x20: Next nonce counter for AES-CCM
	EncryptionMethod   uint32   // 0x24: Encryption method
	CreationTime       uint64   // 0x28: Creation time (Windows FILETIME)
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
	Entries      []FVEMetadataEntry
	PaddingBytes []byte // Padding (seen in Windows 8 metadata blocks)
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

func (volume *Volume) Process(hD readers.DiskReader, volumeStartOffset int64) {
	// Read the first metadata block header (at offset specified in volume header)

	metadataBlockHeaderData, _ := hD.ReadFile(int64(volume.Header.FVEMetadataOffset1)+volumeStartOffset, 64)
	metadataBlockHeader := new(FVEMetadataBlockHeaderV2)
	utils.Unmarshal(metadataBlockHeaderData, metadataBlockHeader)
	volume.MetadataBlocks = append(volume.MetadataBlocks, MetadataBlock{Header: *metadataBlockHeader})
}
