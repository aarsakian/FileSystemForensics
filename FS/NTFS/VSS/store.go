package VSS

var StoreBlockTypes = map[int]string{0x0001: "Volume Header", 0x0002: "Catalog Header", 0x0003: "Block Descriptor List",
	0x0004: "Store Header", 0x0005: "Store Block Ranges list", 0x0006: "Store Bitmap"}

type Store struct {
	Header          *StoreHeader
	BlockLists      []StoreList
	BlockRangeLists []StoreBlockRangeEntry
}

type StoreHeader struct {
	VssGUID        [16]byte
	Version        uint32
	RecordType     uint32
	RelativeOffset uint64
	CurrentOffset  uint64
	NextOffset     uint64
	StoreSize      uint64
	Uknown         [72]byte
}

type StoreList struct {
	OriginalDataBlockOffset      uint64
	RelativeStoreDataBlockOffset uint64
	StoreDataBlockOffset         uint64
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
type StoreBlockRangeEntry struct {
	StartOffset    uint64
	RelativeOffset uint64
	RangeSize      uint64
}
