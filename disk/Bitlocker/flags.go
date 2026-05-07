package bitlocker

// ============================================================================
// Encryption Method Constants
// ============================================================================

const (
	EncryptionMethodUnknown           uint16 = 0x0000 // Unknown or not encrypted
	EncryptionMethodStretchKey1       uint16 = 0x1000 // Stretch key variant 1
	EncryptionMethodStretchKey2       uint16 = 0x1001 // Stretch key variant 2
	EncryptionMethodAESCCM1           uint16 = 0x2000 // AES-CCM 256-bit
	EncryptionMethodAESCCM2           uint16 = 0x2001 // AES-CCM 256-bit variant 1
	EncryptionMethodAESCCM3           uint16 = 0x2002 // AES-CCM 256-bit variant 2
	EncryptionMethodAESCCM4           uint16 = 0x2003 // AES-CCM 256-bit variant 3
	EncryptionMethodAESCCM5           uint16 = 0x2004 // AES-CCM 256-bit variant 4
	EncryptionMethodAESCCM6           uint16 = 0x2005 // AES-CCM 256-bit variant 5
	EncryptionMethodAESCBCElephant128 uint16 = 0x8000 // AES-CBC 128-bit with Elephant Diffuser
	EncryptionMethodAESCBCElephant256 uint16 = 0x8001 // AES-CBC 256-bit with Elephant Diffuser
	EncryptionMethodAESCBC128         uint16 = 0x8002 // AES-CBC 128-bit
	EncryptionMethodAESCBC256         uint16 = 0x8003 // AES-CBC 256-bit
	EncryptionMethodAESXTS128         uint16 = 0x8004 // AES-XTS 128-bit
	EncryptionMethodAESXTS256         uint16 = 0x8005 // AES-XTS 256-bit
)

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
// to be used with Datum ( FVEMetadataHeaderV3 )
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
	MetadataValueTypeAsymEnc       uint16 = 0x000C //Asym
	MetadataValueTypeExportedKey   uint16 = 0x000D //Exported Key
	MetadataValueTypePublicKey     uint16 = 0x000E //public key
	MetadataValueTypeOffsetSize    uint16 = 0x000F // Offset and size (two 64-bit values)
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
