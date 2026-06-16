package datums

import (
	"errors"
	"fmt"

	"github.com/aarsakian/FileSystemForensics/utils"
)

type Datum interface {
	Process([]byte) error
	SetHeader(*DatumHeader)
	GetInfo() string
	GetHeader() *DatumHeader
}

// FVEMetadataEntry represents a single metadata entry.
// Entries are variable-sized and contain different types of data.
type DatumHeader struct {
	EntrySize uint16 // 0x00: Size including this field
	EntryType uint16 // 0x02: Entry type (see MetadataEntryType)
	ValueType uint16 // 0x04: Value type (see MetadataValueType)
	Version   uint16 // 0x06: Version (typically 1, sometimes 3 for clear key VMK)
	Raw       []byte //raw data of header

}

// 0x0001
// FVEKey represents a key with encryption method and key data.
type FVEKey struct {
	Header           *DatumHeader // Common header for all entries
	EncryptionMethod uint32       // Encryption method used
	KeyData          []byte       // Variable-length key data
}

// 0x0002
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
	Nonce         [12]byte     // 12-byte nonce
	EncryptedData []byte       // AES-CCM encrypted data (variable size)

}

// 0x0006
type FVETPMEnc struct {
	Header *DatumHeader
	Uknown uint32
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
	datumHeader.Raw = append([]byte(nil), raw[:8]...)
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
	return fmt.Sprintf("Header info %s FVEKey: Encryption Method: %s Key Data Length: %d bytes",
		key.Header.GetInfo(), GetEncryptionMethod(key.EncryptionMethod), len(key.KeyData))
}

func (key *FVEKey) GetHeader() *DatumHeader {
	return key.Header
}

func (unicodeStr *FVEUnicodeString) Process(raw []byte) error {

	unicodeStr.Value = utils.DecodeUTF16(raw[:])
	return nil
}

func (unicodeStr *FVEUnicodeString) SetHeader(header *DatumHeader) {
	unicodeStr.Header = header
}

func (unicodeStr *FVEUnicodeString) GetHeader() *DatumHeader {
	return unicodeStr.Header
}

func (unicodeStr *FVEUnicodeString) GetInfo() string {
	return fmt.Sprintf("Header info %s FVEUnicodeString: Value: %s", unicodeStr.Header.GetInfo(), unicodeStr.Value)
}

func (stretch *FVEStretchKey) Process(raw []byte) error {
	if _, err := utils.Unmarshal(raw, stretch); err != nil {
		return err
	}
	stretch.EncryptedKey = append([]byte(nil), raw[20:]...)
	return nil
}

func (stretch *FVEStretchKey) SetHeader(header *DatumHeader) {
	stretch.Header = header
}

func (stretch *FVEStretchKey) GetInfo() string {
	return fmt.Sprintf("Header info %s FVEStretchKey: Encryption Method: %s, Salt: %X, Encrypted Key Length: %d bytes",
		stretch.Header.GetInfo(), GetEncryptionMethod(stretch.EncryptionMethod), stretch.Salt, len(stretch.EncryptedKey))
}

func (stretch *FVEStretchKey) GetHeader() *DatumHeader {
	return stretch.Header
}

func (aesccm *FVEAESCCMKey) Process(raw []byte) error {
	copy(aesccm.Nonce[:], raw[:12])
	aesccm.EncryptedData = raw[12:]

	return nil
}

func (aesccm *FVEAESCCMKey) SetHeader(header *DatumHeader) {
	aesccm.Header = header
}

func (aesccm *FVEAESCCMKey) GetMac() []byte {
	return aesccm.EncryptedData[:16]
}

func (aesccm *FVEAESCCMKey) VerifyTag(tag []byte) bool {
	return equal16(aesccm.GetMac(), tag)
}

func (aesccm FVEAESCCMKey) GetNonceTime() string {
	nonceTime := new(utils.WindowsTime)
	utils.Unmarshal(aesccm.Nonce[:12], nonceTime)
	return nonceTime.ConvertToIsoTime()
}

func (aesccm FVEAESCCMKey) GetNonceCounter() uint32 {
	return utils.ToUint32(aesccm.Nonce[12:])
}

func (aesccm *FVEAESCCMKey) GetInfo() string {
	return fmt.Sprintf("Header info %s   Encrypted Data Length: %d bytes Time %s",
		aesccm.Header.GetInfo(), len(aesccm.EncryptedData), aesccm.GetNonceTime())
}

func (aesccm *FVEAESCCMKey) GetHeader() *DatumHeader {
	return aesccm.Header
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

func (external *FVEExternalKey) GetHeader() *DatumHeader {
	return external.Header
}

func (external *FVEExternalKey) GetInfo() string {
	return fmt.Sprintf("Header info %s FVEExternalKey: Properties Length: %d bytes",
		external.Header.GetInfo(), len(external.Properties))
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

func (headerBlock *FVEVolumeHeaderBlock) GetHeader() *DatumHeader {
	return headerBlock.Header
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

func (useKey *FVEUseKey) GetHeader() *DatumHeader {
	return useKey.Header
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
		return fmt.Sprintf("unknown type %x", dh.EntryType)
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

func GetEncryptionMethod(encryptionMethod uint32) string {
	//remove high 16 bits which may contain flags
	switch encryptionMethod & 0xFFFF {
	case EncryptionMethodStretchKey1:
		return "Recovery key stretching variant 2"
	case EncryptionMethodStretchKey2:
		return "Password stretching variant 2"
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
		return "AES-XTS 256-bit"
	}
	return fmt.Sprintf("Unknown encryption method: 0x%08x", encryptionMethod)
}

func equal16(a, b []byte) bool {
	if len(a) != 16 || len(b) != 16 {
		return false
	}
	var v byte
	for i := 0; i < 16; i++ {
		v |= a[i] ^ b[i]
	}
	return v == 0
}
