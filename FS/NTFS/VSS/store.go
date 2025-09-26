package VSS

import (
	"github.com/aarsakian/FileSystemForensics/utils"
)

var StoreBlockTypes = map[int]string{0x0001: "Volume Header", 0x0002: "Catalog Header", 0x0003: "Block Descriptor List",
	0x0004: "Store Header", 0x0005: "Store Block Ranges list", 0x0006: "Store Bitmap"}

var DataBlockTypes = map[int]string{0x00000001: "Forwarder", 0x00000002: "Overlay", 0x00000004: "Not Used"}

type Store struct {
	Header          *StoreHeader
	Info            *StoreInfo
	StoresList      *StoreList
	StoreBlockRange *StoreBlockRange
	BitmapData      []byte
	PrevBitmapData  []byte
}

// System Restore Metadata
type RSTR struct {
	Signature [4]byte
	Version   uint32
	GUID      [16]byte
}

type StoreHeader struct {
	VssGUID        [16]byte
	Version        uint32
	RecordType     uint32
	RelativeOffset uint64 //from start of store
	CurrentOffset  uint64 //from start of store
	NextOffset     uint64 //from start of volume 0-> last block
	StoreSize      uint64 //0 except first block header
	Uknown         [72]byte
}

type StoreList struct {
	Header           *StoreHeader
	BlockDescriptors []StoreBlockDescriptor
}

type StoreBlockDescriptor struct {
	OriginalDataBlockOffset      uint64 //from volume
	RelativeStoreDataBlockOffset uint64
	StoreDataBlockOffset         uint64 //from volume
	Flags                        [4]byte
	AllocationBitmap             [4]byte
}

type StoreInfo struct {
	UknownGUID               [16]byte
	ShadowCopyGUID           [16]byte
	ShadowCopySetGUID        [16]byte
	SnapshotContext          [4]byte
	Uknown1                  [4]byte
	AttributeFlags           [4]byte
	Uknown2                  [4]byte
	OperatingMachineNameSize uint16
	OperatingMachineName     string
	ServiceMachineNameSize   uint16
	ServiceMachineName       string
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

type StoreBlockRange struct {
	Header          *StoreHeader
	BlockRangeLists []StoreBlockRangeEntry
}

type StoreBlockRangeEntry struct {
	StartOffset    uint64 //from the start of volume
	RelativeOffset uint64 //from the start of store
	RangeSize      uint64
}

func (storeBlockRangeEntry StoreBlockRangeEntry) LocateClusters(clusters []int) {

}
