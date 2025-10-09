package VSS

import (
	"fmt"

	"github.com/aarsakian/FileSystemForensics/utils"
)

// RCRD recorded Redirected block
var StoreBlockTypes = map[uint32]string{0x0001: "Volume Header", 0x0002: "Catalog Header", 0x0003: "Block Descriptor List",
	0x0004: "Store Header", 0x0005: "Store Block Ranges list", 0x0006: "Store Bitmap"}

var DataBlockTypes = map[int]string{0x00000001: "Forwarder",
	0x00000002: "Overlay", 0x00000004: "Not Used"}

var StoreBlockFlags = map[uint32]string{
	0x00000001: "Allocated",         // Block contains valid data
	0x00000002: "Compressed",        // Block is compressed
	0x00000004: "Sparse",            // Block is sparse (zero-filled)
	0x00000008: "Encrypted",         // Block is encrypted
	0x00000010: "ChecksumProtected", // Block has checksum for integrity
	0x00000020: "DeltaEncoded",      // Block stores differences from previous version
	0x00000040: "CatalogReferenced", // Block is referenced in snapshot catalog
}

var StoreAttributeFlags = map[uint32]string{
	0x00000001: "ReadOnly",
	0x00000002: "Hidden",
	0x00000004: "System",
	0x00000008: "Temporary",
	0x00000010: "SnapshotData",
	0x00000020: "Metadata",
	0x00000040: "Catalog",
	0x00000080: "Deleted",
}

type Store struct {
	Header          *StoreHeader
	Info            *StoreInfo
	StoresList      *StoreList
	StoreBlockRange *StoreBlockRange
	BitmapData      []byte
	PrevBitmapData  []byte
}

type StoreHeader struct {
	VssGUID       [16]byte
	Version       uint32
	RecordType    uint32
	ShadowOffset  uint64 //from start of store
	CurrentOffset uint64 //from start of store
	NextOffset    uint64 //from start of volume 0-> last block
	StoreSize     uint64 //0 except first block header
	Uknown        [72]byte
}

type StoreList struct {
	Header           *StoreHeader
	BlockDescriptors []StoreBlockDescriptor
}

type StoreBlockDescriptor struct {
	OriginalDataBlockOffset uint64 //from volume
	ShadowDataBlockOffset   uint64 //shadow offset
	StoreDataBlockOffset    uint64 //from volume
	Flags                   uint32
	AllocationBitmap        [4]byte
}

type StoreInfo struct {
	UknownGUID               [16]byte
	ShadowCopyGUID           [16]byte
	ShadowCopySetGUID        [16]byte
	SnapshotContext          [4]byte
	Uknown1                  [4]byte
	AttributeFlags           uint32
	Uknown2                  [4]byte
	OperatingMachineNameSize uint16
	OperatingMachineName     string
	ServiceMachineNameSize   uint16
	ServiceMachineName       string
}

type StoreBlockRange struct {
	Header          *StoreHeader
	BlockRangeLists []StoreBlockRangeEntry
}

type StoreBlockRangeEntry struct {
	StartOffset  uint64 //from the start of volume
	ShadowOffset uint64 //from the start of store
	RangeSize    uint64
}

func (storeHeader StoreHeader) GetInfo() string {
	return utils.StringifyGUID(storeHeader.VssGUID[:]) + " V:" + fmt.Sprintf("%d", storeHeader.Version) +
		" Record Type:" + StoreBlockTypes[storeHeader.RecordType]
}

func (store *Store) Process(data []byte) int {

	storeInfoHeader := new(StoreHeader)

	readBytes, _ := utils.Unmarshal(data, storeInfoHeader)
	offset := readBytes

	storeInfo := new(StoreInfo)
	readBytes, _ = utils.Unmarshal(data[offset:], storeInfo)
	offset = readBytes - 2 //workaround it counts ServiceMachineNameSize

	storeInfo.OperatingMachineName = utils.DecodeUTF16(data[offset : offset+int(storeInfo.OperatingMachineNameSize)])
	offset += int(storeInfo.OperatingMachineNameSize)

	storeInfo.ServiceMachineNameSize = utils.ToUint16(data[offset:])
	offset += 2
	storeInfo.ServiceMachineName = utils.DecodeUTF16(data[offset : offset+int(storeInfo.ServiceMachineNameSize)])

	store.Header = storeInfoHeader
	store.Info = storeInfo
	return offset
}

func (storeInfo StoreInfo) DecodeStoreAttributeFlags() string {
	var result string
	for bit, label := range StoreAttributeFlags {
		if storeInfo.AttributeFlags&bit != 0 {
			result += " " + label
		}
	}
	return result
}

func (storeBlockDescriptor StoreBlockDescriptor) DecodeStoreBlockFlags() string {
	var result string
	for bit, label := range StoreBlockFlags {
		if storeBlockDescriptor.Flags&bit != 0 {
			result += " " + label
		}
	}
	return result
}

func (storeBlockRangeEntry StoreBlockRangeEntry) HasClusters(cluster int) bool {
	if storeBlockRangeEntry.StartOffset < uint64(cluster*4096) &&
		uint64(cluster*4096) < storeBlockRangeEntry.StartOffset+storeBlockRangeEntry.RangeSize {
		return true
	}
	return false
}
