package bitlocker

import (
	"crypto/aes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"
	"unicode/utf16"

	"github.com/aarsakian/FileSystemForensics/disk/Bitlocker/datums"
	ccm "github.com/aarsakian/FileSystemForensics/disk/Bitlocker/decryption"
	"github.com/aarsakian/FileSystemForensics/logger"
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
// High-level Volume Structure
// ============================================================================

// Volume represents a BitLocker encrypted volume (supports Windows 7+).
// For Windows Vista, use alternate parsing logic.
type Volume struct {
	Header         VolumeHeaderWindows7 // Works for Windows 7+ (includes Windows 8/10)
	MetadataBlocks []MetadataBlock
	Key            []byte // Decrypted FVEK (File Volume Encryption Key) after decryption
}

//========================================================================
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

func (volume Volume) GetInfo() {
	fmt.Printf("Volume INFO %s\n", utils.StringifyGUID(volume.Header.BitLockerID[:]))
	for _, metadataBlock := range volume.MetadataBlocks {
		metadataBlock.GetInfo()
	}

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

	return nil
}

func DeriveKeyFromRecoveryKey(key string) ([]byte, error) {
	derivedKey := make([]byte, 16)
	for groupId, group := range strings.Split(key, "-") {

		utf16Units := utf16.Encode([]rune(group))
		value := 0
		for i, cu := range utf16Units {
			if cu < '0' || cu > '9' {
				msg := fmt.Sprintf("invalid digit %d at index %d", cu, i)
				return nil, errors.New(msg)
			}
			//code unit -> numeric digit
			digit := int(cu - '0')
			//accumulate
			value = value*10 + digit
		}
		if value%11 != 0 {
			break
		}
		value /= 11
		if value > math.MaxUint16 {
			break
		}
		binary.LittleEndian.PutUint16(derivedKey[groupId*2:groupId*2+2], uint16(value))

	}
	return derivedKey, nil

}

func StretchKey(initialHash []byte, salt []byte, iterations int) []byte {
	var lastHash [32]byte
	saltArray := [16]byte{}
	copy(saltArray[:], salt)

	buf := make([]byte, 88)

	copy(buf[32:64], initialHash[:])
	copy(buf[64:80], saltArray[:])

	for i := uint64(0); i < uint64(iterations); i++ {
		binary.LittleEndian.PutUint64(buf[80:88], uint64(i))
		lastHash = sha256.Sum256(buf[:])
		copy(buf[:32], lastHash[:])

	}

	return lastHash[:]
}

func (volume *Volume) Decrypt(password string, recoverykey string) error {

	vmkKey, err := volume.DecryptVMK(password, recoverykey)
	if err != nil {
		return err
	}
	version := utils.ToUint16(vmkKey[20:])
	dataSize := utils.ToUint16(vmkKey[16:])
	counter := int(vmkKey[24])
	if version == 1 && dataSize == 44 {

		vmkKey = vmkKey[28:]

	}
	candidateFVEKeys, err := volume.DecryptFVEK(vmkKey)
	if err != nil {
		return err
	}
	for _, fvekKey := range candidateFVEKeys {
		version = utils.ToUint16(fvekKey[20:])
		dataSize = utils.ToUint16(fvekKey[16:])
		fvecounter := int(fvekKey[24])
		if version == 1 && dataSize == 44 && fvecounter > counter {

			fvekKey = fvekKey[28:]
			fmt.Printf("Decrypted fve key %x\n", fvekKey)
			volume.Key = fvekKey
			return nil
		}
	}

	return errors.New("failed to decrypt FVEK with provided password or recovery key")

}

func (volume Volume) DecryptVMK(password string, recoverykey string) ([]byte, error) {
	var key []byte
	for _, metadataBlock := range volume.MetadataBlocks {
		for _, datum := range metadataBlock.Datums {
			vmk, ok := datum.(*datums.FVEVolumeMasterKey)
			if ok {

				for _, vmkDatum := range vmk.Datums {
					if fvekey, ok := vmkDatum.(*datums.FVEKey); ok {

						if datums.GetEncryptionMethod(fvekey.EncryptionMethod[:]) != "AES-CCM 256-bit" {
							return nil, errors.New("unsupported VMK encryption method")
						}
						if vmk.GetProtectionType() == "Clear Key" {
							key = fvekey.KeyData
						}
					} else if stretchkey, ok := vmkDatum.(*datums.FVEStretchKey); ok {
						if datums.GetEncryptionMethod(stretchkey.EncryptionMethod[:]) ==
							"Password stretching variant 2" && password != "" {

							initialHash := sha256.Sum256(utils.PasswordUTF16LE(password))
							initialHash = sha256.Sum256(initialHash[:])
							key = StretchKey(initialHash[:], stretchkey.Salt[:], 0x100000)

						} else if datums.GetEncryptionMethod(stretchkey.EncryptionMethod[:]) ==
							"Recovery key stretching variant 2" && recoverykey != "" {
							stretchedKey, err := DeriveKeyFromRecoveryKey(recoverykey)
							if err != nil {
								return nil, err
							}
							initialHash := sha256.Sum256(stretchedKey)
							key = StretchKey(initialHash[:], stretchkey.Salt[:], 0x100000)

						}
					} else if aesccmKey, ok := vmkDatum.(*datums.FVEAESCCMKey); ok {
						//cannot decrypt
						if key == nil {
							continue
						}
						aad := vmkDatum.GetHeader().Raw
						msg := fmt.Sprintf("key: %x aesccmKey.EncryptedData[:] %x nonce %x aad %x mac %x\n",
							key, aesccmKey.EncryptedData[:], aesccmKey.Nonce[:], aad, aesccmKey.GetMac())
						decryptedVMKKey, err := ccm.OpenCCM(key, aesccmKey.EncryptedData[:], aesccmKey.Nonce[:])
						logger.FSLogger.Info(msg)

						if err != nil {
							return nil, err
						}
						expectedTag, err := ccm.GetExpectedTag(key, aesccmKey.Nonce[:], aad, decryptedVMKKey[16:], aesccmKey.GetMac())
						if err != nil {
							return nil, err
						}

						aesccmKey.VerifyTag(expectedTag)
						return decryptedVMKKey, nil
					}
				}
			}
		}
	}
	return nil, errors.New("VMK key not found")
}

func (volume Volume) DecryptFVEK(vmkKey []byte) ([][]byte, error) {
	candidateKeys := [][]byte{vmkKey}
	for _, datum := range volume.MetadataBlocks[0].Datums {

		// Skip VMK entry entirely
		if _, isVMK := datum.(*datums.FVEVolumeMasterKey); isVMK {
			continue
		}

		if aesccmKey, ok := datum.(*datums.FVEAESCCMKey); ok {

			decryptedFVEK, err := ccm.OpenCCM(vmkKey, aesccmKey.EncryptedData[:], aesccmKey.Nonce[:])
			if err != nil {
				return nil, err
			}
			candidateKeys = append(candidateKeys, decryptedFVEK)
		}
	}

	return candidateKeys, nil
}

func (volume Volume) GetEncryptionMethod() string {
	return datums.GetEncryptionMethod(volume.MetadataBlocks[0].MetadataV1.EncryptionMethod[:])
}

func (volume Volume) GetVolumeOffset() int64 {
	return int64(volume.MetadataBlocks[0].Header.VolumeHeaderOffset)
}

func (volume Volume) DecryptData(sectorOffset int) error {
	blockKey := make([]byte, 16)
	binary.LittleEndian.PutUint64(blockKey, uint64(uint16(sectorOffset)/volume.Header.BytesPerSector))
	IV := make([]byte, 16)

	sectorKey := make([]byte, 32)

	/* uint8_t sector_key_data[ 32 ];*/
	if volume.GetEncryptionMethod() == "AES-CBC 128-bit with Elephant Diffuser" ||
		volume.GetEncryptionMethod() == "AES-CBC 256-bit with Elephant Diffuser" ||
		volume.GetEncryptionMethod() == "AES-CBC 128-bit" ||
		volume.GetEncryptionMethod() == "AES-CBC 256-bit" {

		block, err := aes.NewCipher(volume.Key)
		if err != nil {
			return err
		}
		// Use IV for decryption of sector data with AES-CBC or AES-CBC with Elephant Diffuser
		// Decrypt the sector data using the derived IV and the FVEK
		block.Encrypt(IV, blockKey)

		if volume.GetEncryptionMethod() == "AES-CBC 128-bit with Elephant Diffuser" ||
			volume.GetEncryptionMethod() == "AES-CBC 256-bit with Elephant Diffuser" {

			block.Encrypt(sectorKey, blockKey)
			blockKey[15] = 0x80

			block.Encrypt(sectorKey[16:], blockKey)
		}
	} else if volume.GetEncryptionMethod() == "AES-XTS 128-bit" ||
		volume.GetEncryptionMethod() == "AES-XTS 256-bit" {
		copy(IV, blockKey)

	}
	return nil

}
