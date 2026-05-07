package bitlocker

import (
	"errors"
	"fmt"

	"github.com/aarsakian/FileSystemForensics/utils"
)

type Datum interface {
	Process([]byte) error
	SetHeader(*DatumHeader)
	GetInfo() string
}

// FVEMetadataEntry represents a single metadata entry.
// Entries are variable-sized and contain different types of data.
type DatumHeader struct {
	EntrySize uint16 // 0x00: Size including this field
	EntryType uint16 // 0x02: Entry type (see MetadataEntryType)
	ValueType uint16 // 0x04: Value type (see MetadataValueType)
	Version   uint16 // 0x06: Version (typically 1, sometimes 3 for clear key VMK)

}

// FVEKey represents a key with encryption method and key data.
type FVEKey struct {
	Header           *DatumHeader // Common header for all entries
	EncryptionMethod uint32       // Encryption method used
	KeyData          []byte       // Variable-length key data
}

type FVEUnicodeString struct {
	Header *DatumHeader // Common header for all entries
	Value  string       // UTF-16 LE encoded string with null terminator
}

// FVEStretchKey is used for password/recovery key protection.
// Value type: 0x0003
type FVEStretchKey struct {
	Header           *DatumHeader // Common header for all entries
	EncryptionMethod uint32       // Encryption method
	Salt             [16]byte     // 16-byte salt
	EncryptedKey     []byte       // AES-CCM encrypted key follows (variable size)
}

// Value type: 0x0004
type FVEUseKey struct {
	Header    *DatumHeader
	Algorithm uint16
	Padding   uint16
}

// FVEAESCCMKey represents an AES-CCM encrypted key entry.
// Value type: 0x0005
type FVEAESCCMKey struct {
	Header        *DatumHeader // Common header for all entries
	NonceTime     utils.WindowsTime
	NonceCounter  uint32
	EncryptedData []byte // AES-CCM encrypted data (variable size)
}

// 0x0006
type FVETPMEnc struct {
	Header *DatumHeader
	Uknown uint32
}

// FVEVolumeMasterKey represents the VMK entry.
// Value type: 0x0008
type FVEVolumeMasterKey struct {
	Header               *DatumHeader      // Common header for all entries
	KeyIdentifier        [16]byte          // GUID identifying this VMK
	LastModificationTime utils.WindowsTime // FILETIME of last modification
	Unknown              uint16            // Unknown field
	ProtectionType       uint16            // Protection type (see VMKProtectionType constants)
	Properties           []byte            // Variable array of property entries (entry type 0x0000)
}

// FVEExternalKey represents a startup key (USB key) entry.
// Value type: 0x0009
type FVEExternalKey struct {
	Header               *DatumHeader      // Common header for all entries
	KeyIdentifier        [16]byte          // GUID identifying this key
	LastModificationTime utils.WindowsTime // FILETIME of last modification
	Properties           []byte            // Variable array of property entries
}

// FVEVolumeHeaderBlock specifies the location of the encrypted volume header.
// Value type: 0x000F
type FVEVolumeHeaderBlock struct {
	Header         *DatumHeader
	BlockOffset    uint64 // Offset where encrypted volume header is stored
	BlockSize      uint64 // Size of encrypted volume header in bytes
	Unknown1       uint16 // Unknown (entry count?)
	Unknown2       uint16 // Unknown (size of additional data?)
	AdditionalData []byte // Variable additional data (array of 14-byte entries)
}

func (datumHeader *DatumHeader) Process(raw []byte) error {
	if len(raw) < 8 {
		return errors.New("metadata entry buffer too small")
	}

	if _, err := utils.Unmarshal(raw[:8], datumHeader); err != nil {
		return err
	}
	if datumHeader.EntrySize < 8 {
		return errors.New("invalid metadata entry size")
	}
	if int(datumHeader.EntrySize) > len(raw) {
		return errors.New("metadata entry extends beyond buffer")
	}

	return nil
}

func (key *FVEKey) Process(raw []byte) error {
	if len(raw) < 32 {
		return errors.New("FVEKey buffer too small")
	}
	if _, err := utils.Unmarshal(raw, key); err != nil {
		return err
	}
	key.KeyData = append([]byte(nil), raw[4:]...)
	return nil
}

func (key *FVEKey) SetHeader(header *DatumHeader) {
	key.Header = header
}

func (key *FVEKey) GetInfo() string {
	return fmt.Sprintf("Header info %s FVEKey: Encryption Method: 0x%08X, Key Data Length: %d bytes",
		key.Header.GetInfo(), key.EncryptionMethod, len(key.KeyData))
}

func (unicodeStr *FVEUnicodeString) Process(raw []byte) error {

	unicodeStr.Value = utils.DecodeUTF16(raw[:])
	return nil
}

func (unicodeStr *FVEUnicodeString) SetHeader(header *DatumHeader) {
	unicodeStr.Header = header
}

func (unicodeStr *FVEUnicodeString) GetInfo() string {
	return fmt.Sprintf("Header info %s FVEUnicodeString: Value: %s", unicodeStr.Header.GetInfo(), unicodeStr.Value)
}

func (stretch *FVEStretchKey) Process(raw []byte) error {
	if _, err := utils.Unmarshal(raw, &stretch); err != nil {
		return err
	}
	stretch.EncryptedKey = append([]byte(nil), raw[20:]...)
	return nil
}

func (stretch *FVEStretchKey) SetHeader(header *DatumHeader) {
	stretch.Header = header
}

func (stretch *FVEStretchKey) GetInfo() string {
	return fmt.Sprintf("Header info %s FVEStretchKey: Encryption Method: 0x%08X, Salt: %X, Encrypted Key Length: %d bytes",
		stretch.Header.GetInfo(), stretch.EncryptionMethod, stretch.Salt, len(stretch.EncryptedKey))
}

func (aesccm *FVEAESCCMKey) Process(raw []byte) error {
	if _, err := utils.Unmarshal(raw, aesccm); err != nil {
		return err
	}
	aesccm.EncryptedData = append([]byte(nil), raw[12:]...)
	return nil
}

func (aesccm *FVEAESCCMKey) SetHeader(header *DatumHeader) {
	aesccm.Header = header
}

func (aesccm *FVEAESCCMKey) GetInfo() string {
	return fmt.Sprintf("Header info %s   Encrypted Data Length: %d bytes Time %s",
		aesccm.Header.GetInfo(), len(aesccm.EncryptedData), aesccm.NonceTime.ConvertToIsoTime())
}

func (vmk *FVEVolumeMasterKey) Process(raw []byte) error {
	if _, err := utils.Unmarshal(raw, vmk); err != nil {
		return err
	}
	vmk.Properties = append([]byte(nil), raw[28:]...)
	return nil
}

func (vmk *FVEVolumeMasterKey) SetHeader(header *DatumHeader) {
	vmk.Header = header
}

func (vmk *FVEVolumeMasterKey) GetInfo() string {
	return fmt.Sprintf("Header info %s FVEVolumeMasterKey: Properties Length: %d bytes Protection Type %s Time %s",
		vmk.Header.GetInfo(), len(vmk.Properties), vmk.GetProtectionType(),
		vmk.LastModificationTime.ConvertToIsoTime())
}

func (external *FVEExternalKey) Process(raw []byte) error {
	if _, err := utils.Unmarshal(raw, external); err != nil {
		return err
	}
	external.Properties = append([]byte(nil), raw[24:]...)

	return nil
}

func (external *FVEExternalKey) SetHeader(header *DatumHeader) {
	external.Header = header
}

func (external *FVEExternalKey) GetInfo() string {
	return fmt.Sprintf("Header info %s FVEExternalKey: Properties Length: %d bytes", external.Header.GetInfo(), len(external.Properties))
}

func (headerBlock *FVEVolumeHeaderBlock) Process(raw []byte) error {
	if _, err := utils.Unmarshal(raw, headerBlock); err != nil {
		return err
	}
	headerBlock.AdditionalData = append([]byte(nil), raw[20:]...)

	return nil
}

func (headerBlock *FVEVolumeHeaderBlock) SetHeader(header *DatumHeader) {
	headerBlock.Header = header
}

func (headerBlock *FVEVolumeHeaderBlock) GetInfo() string {
	return fmt.Sprintf("Header info %s FVEVolumeHeaderBlock: Block Offset: %d, Block Size: %d, Additional Data Length: %d bytes",
		headerBlock.Header.GetInfo(), headerBlock.BlockOffset, headerBlock.BlockSize, len(headerBlock.AdditionalData))
}

func (useKey *FVEUseKey) Process(raw []byte) error {
	if _, err := utils.Unmarshal(raw, useKey); err != nil {
		return err
	}
	return nil
}

func (useKey *FVEUseKey) SetHeader(header *DatumHeader) {
	useKey.Header = header
}

func (useKey FVEUseKey) GetInfo() string {
	return fmt.Sprintf("algo %d", useKey.Algorithm)
}

func CreateDatum(dh DatumHeader, raw []byte) (Datum, error) {
	var datum Datum
	switch dh.ValueType {
	case MetadataValueTypeErased:
		return nil, nil
	case MetadataValueTypeKey:
		datum = new(FVEKey)

	case MetadataValueTypeUnicodeString:
		datum = new(FVEUnicodeString)

	case MetadataValueTypeStretchKey:
		datum = new(FVEStretchKey)

	case MetadataValueTypeAESCCMKey:
		datum = new(FVEAESCCMKey)

	case MetadataValueTypeVMK:
		datum = new(FVEVolumeMasterKey)

	case MetadataValueTypeUseKey:
		datum = new(FVEUseKey)

	case MetadataValueTypeExternalKey:
		datum = new(FVEExternalKey)

	case MetadataValueTypeOffsetSize:
		datum = new(FVEVolumeHeaderBlock)

	}
	if datum == nil {
		return nil, fmt.Errorf("unsupported value type: 0x%04X", dh.ValueType)
	}
	datum.SetHeader(&dh)
	datum.Process(raw)
	return datum, nil
}

func (dh DatumHeader) GetInfo() string {
	return fmt.Sprintf("Entry Type: %s, Value Type: %s", dh.GetEntryType(), dh.GetValueType())
}

func (dh DatumHeader) GetEntryType() string {
	switch dh.EntryType {
	case MetadataEntryTypeVMK:
		return "Volume Master Key (VMK) entry"
	case MetadataEntryTypeFVEK:
		return "Full Volume Encryption Key (FVEK) entry"
	case MetadataEntryTypeValidation:
		return "Validation data entry"
	case MetadataEntryTypeStartupKey:
		return "Startup key (external key) entry"
	case MetadataEntryTypeDescription:
		return "Description (drive label) entry"
	case MetadataEntryTypeFVEKBackup:
		return "Backup FVEK entry"
	case MetadataEntryTypeVolumeHeader:
		return "Volume header block entry"
	default:
		return "Unknown entry type"
	}
}

func (dh DatumHeader) GetValueType() string {
	switch dh.ValueType {
	case MetadataValueTypeErased:
		return "Erased entry"
	case MetadataValueTypeKey:
		return "Encryption key"
	case MetadataValueTypeUnicodeString:
		return "Unicode string (UTF-16 LE with null terminator)"
	case MetadataValueTypeStretchKey:
		return "Stretch key (salt + AES-CCM encrypted key)"
	case MetadataValueTypeUseKey:
		return "Use key"
	case MetadataValueTypeAESCCMKey:
		return "AES-CCM encrypted key"
	case MetadataValueTypeTPMEncodedKey:
		return "TPM encoded key"
	case MetadataValueTypeValidation:
		return "Validation"
	case MetadataValueTypeVMK:
		return "Volume Master Key structure"
	case MetadataValueTypeExternalKey:
		return "External key"
	case MetadataValueTypeUpdate:
		return "Update entry"
	case MetadataValueTypeError:
		return "Error entry"
	case MetadataValueTypeOffsetSize:
		return "Offset and size (two 64-bit values)"
	default:
		return "Unknown value type"
	}
}

func (vmk FVEVolumeMasterKey) GetProtectionType() string {
	switch vmk.ProtectionType {
	case VMKProtectionTypeClearKey:
		return "Clear Key"
	case VMKProtectionTypeTPM:
		return "TPM"
	case VMKProtectionTypeStartupKey:
		return "Startup Key"
	case VMKProtectionTypeTPMPIN:
		return "TPMIN"
	case VMKProtectionTypeRecoveryPW:
		return "Recovery PW"
	case VMKProtectionTypePassword:
		return "Password"
	default:
		return "Unknown value type"
	}
}
