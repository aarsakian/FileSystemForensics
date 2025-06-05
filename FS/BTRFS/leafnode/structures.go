package leafnode

import (
	"fmt"

	"github.com/aarsakian/FileSystemForensics/logger"
	"github.com/aarsakian/FileSystemForensics/utils"
)

/*
Key meaning
InodeItem objectid = inode number
InodeItem offset = 0
RootItem objectid = 1-9 or subvolume id
RootItem offset = 0 when TREE_OBJECTID type or subvolume otherwise !=0 snapshot id
BlockGroup objectid = starting logical address of chunk
FileExtent objectid = inode number of the file this extent describes
ExtentData objectid =  logical address of the extent
ExtentData offset = starting offset within the file
DirItem objectid = inode of directory containing this entry
DirItem offset  = crc32c name hash of either the direntry name (user visible) or the name
of the extended attribute name
DirIndex objectid = inode number of directory we are looking up into
DirIndex offset = index id in the dir (starts at 2 due to '.', '..')
InodeRef objectid = inode number of file
InodeRef offset = inode number of parent of file
ExtentedInodRef objectid = inode number of file
*/

type Key struct {
	ObjectID uint64
	ItemType uint8
	Offset   uint64
}

type LeafNode struct {
	Items     []Item
	DataItems []DataItem
}

type DataItem interface {
	Parse([]byte) int
	ShowInfo()
	GetInfo() string
}

type Item struct {
	Key        *Key
	DataOffset uint32 //relative to the end of the header
	DataLen    uint32
}

type ExtentItem struct {
	RefCount         uint64
	Generation       uint64
	Flags            uint64 //contain data or metadata
	InlineRef        *InlineRef
	ExtentDataRef    *ExtentDataRef
	ExtentDataRefKey *ExtentDataRefKey
}

type ExtentData struct {
	Generation     uint64
	LogicalDataLen uint64 //size of decoded text
	Compression    uint8
	Encryption     uint8
	OtherEncoding  uint16
	Type           uint8
	ExtentDataRem  *ExtentDataRem
	InlineData     []byte
}

type ExtentDataRef struct {
	Root     uint64
	ObjectID uint64
	Offset   uint64
	Count    uint32
}

type ExtentDataRefKey struct {
	Count uint32
}

// used when no compression, encryption other encoding is used non inline
type ExtentDataRem struct {
	LogicaAddress uint64 //logical address of extent
	Size          uint64
	Offset        uint64
	LogicalBytes  uint64
}

type InlineRef struct {
	Type   uint8 //extent is shared or not
	Offset uint64
}

// c BTRFS_DEV_ITEM_KEYs which help map physical offsets to logical offsets
type DevItem struct { //98B
	DeviceID           uint64
	NofBytes           uint64
	NofUsedBytes       uint64
	OptimalIOAlignment uint32
	OptimalIOWidth     uint32
	MinimailIOSize     uint32
	Type               uint64
	Generation         uint64
	StartOffset        uint64
	DevGroup           uint32
	SeekSpeed          uint8
	Bandwidth          uint8
	DeviceUUID         [16]byte
	FilesystemUUID     [16]byte
}

// used to describe the logical address spac
type ChunkItem struct { //46 byte
	Size               uint64
	Owner              uint64
	StripeLen          uint64
	Type               uint64
	OptimalIOAlignment uint32
	OptimalIOWidth     uint32
	MinimalIOSize      uint32 //sector size
	NofStripes         uint16
	NofSubStripes      uint16
	Stripes            ChunkItemStripes
}

type RootItem struct {
	Inode        *InodeItem
	Generation   uint64
	RootDirID    uint64 //256 for subvolumes 0 other cases
	Bytenr       uint64 //block number of the root node
	ByteLimit    uint64
	BytesUsed    uint64
	LastSnapshot uint64
	Flags        uint64
	Refs         uint32
	DropProgress *Key //60
	DropLevel    uint8
	Level        uint8
	GenV2        uint64
	Uuid         [16]byte
	ParentUuid   [16]byte
	ReceivedUuid [16]byte
	CtransID     uint64
	OtransID     uint64
	StransID     uint64
	RtransID     uint64         //135
	ATime        utils.TimeSpec //?
	CTime        utils.TimeSpec
	MTime        utils.TimeSpec
	OTime        utils.TimeSpec //193
}

// 160B
type InodeItem struct {
	Generation uint64
	Transid    uint64
	StSize     uint64
	StBlock    uint64
	BlockGroup uint64
	StNlink    uint32
	StUid      uint32
	StGid      uint32
	StMode     uint32
	StRdev     uint64
	Flags      uint64
	Sequence   uint64
	Reserved   [32]byte //112
	ATime      utils.TimeSpec
	CTime      utils.TimeSpec
	MTime      utils.TimeSpec
	OTime      utils.TimeSpec
}

// size 18
// used to identify snapshot or volume
type RootRef struct {
	DirID  uint64
	Index  uint64
	Length uint16
	Name   string
}

// 26
type RootBackRef struct {
	DirID  uint64
	Index  uint64
	Length uint16
	Name   string
}

// name entry for inode
type InodeRef struct {
	Index  uint64
	Length uint16
	Name   string
}

type ChunkItemStripes []ChunkItemStripe

type ChunkItemStripe struct {
	DeviceID   uint64
	Offset     uint64
	DeviceUUID [16]byte
}

type DevExtentItem struct {
	ChunkTree     uint64
	ChunkObjectID uint64
	ChunkOffset   uint64
	Length        uint64
	UUID          [16]byte
}

type DevStatsItem struct {
}

// allow lookup by name
type DirItem struct {
	Transid   uint64
	DataLen   uint16 //Xattr Len 0 otherwise
	NameLen   uint16 //Xattr Name len or dir len
	Type      uint8
	Name      string //name of ordinary dir entry or xattr name
	XattValue string
}

// Dir Index items always contain one entry and their key
// is the index of the file in the current directory
type DirIndex struct {
	Transid uint64
	DataLen uint16
	NameLen uint16
	Type    uint8
	Name    string
}

type BlockGroupItem struct {
	Used          uint64
	ChunkObjectID uint64
	Flags         uint64
}

// every 4kb of every file written contains checksum
type ChecksumItem struct {
	Checksums []uint32 //crc32
}

func (key *Key) Parse(data []byte) int {
	offset, _ := utils.Unmarshal(data, key)
	return offset
}

func (key Key) ShowInfo() {
	fmt.Printf("Object id %d type %s type %s offset id %d \n", key.ObjectID, ObjectTypes[int(key.ObjectID)],
		ItemTypes[int(key.ItemType)], key.Offset)
}

func (key Key) GetType() string {
	return ItemTypes[int(key.ItemType)]
}

func (checkSumItem *ChecksumItem) Parse(data []byte) int {
	offset := 0

	for offset < len(data) {
		checkSumItem.Checksums = append(checkSumItem.Checksums, utils.ToUint32(data[offset:offset+4]))
		offset += 4
	}
	return offset
}

func (checkSumItem ChecksumItem) ShowInfo() {

}

func (checkSumItem ChecksumItem) GetInfo() string {
	return ""
}

func (blockGroupItem *BlockGroupItem) Parse(data []byte) int {
	offset, _ := utils.Unmarshal(data, blockGroupItem)
	return offset
}

func (blockGroupItem BlockGroupItem) GetInfo() string {
	return ""
}

func (blockGroupItem BlockGroupItem) ShowInfo() {

}

func (devStats *DevStatsItem) Parse(data []byte) int {
	return len(data)
}

func (devStats DevStatsItem) GetInfo() string {
	return ""
}

func (devStats DevStatsItem) ShowInfo() {

}

func (devExtentItem *DevExtentItem) Parse(data []byte) int {
	offset, _ := utils.Unmarshal(data, devExtentItem)
	return offset
}

func (devExtentItem DevExtentItem) ShowInfo() {

}

func (devExtentItem DevExtentItem) GetInfo() string {
	return ""
}

func (rootRef *RootRef) Parse(data []byte) int {
	offset, _ := utils.Unmarshal(data, rootRef)
	if int(rootRef.Length) < offset { //error in parsing?
		rootRef.Name = string(data[offset : offset+int(rootRef.Length)])
		return offset + int(rootRef.Length)
	} else {
		return offset
	}

}

func (rootRef RootRef) ShowInfo() {

}

func (rootRef RootRef) GetInfo() string {
	return ""
}

func (rootBackRef *RootBackRef) Parse(data []byte) int {
	offset, _ := utils.Unmarshal(data, rootBackRef)
	if int(rootBackRef.Length) < offset { //error in parsing?
		rootBackRef.Name = string(data[offset : offset+int(rootBackRef.Length)])
		return offset + int(rootBackRef.Length)
	} else {
		return offset
	}
}

func (rootBackRef RootBackRef) GetInfo() string {
	return ""
}

func (rootBackRef RootBackRef) ShowInfo() {

}

func (dirIndex *DirIndex) Parse(data []byte) int {
	offset, _ := utils.Unmarshal(data, dirIndex)
	if offset+int(dirIndex.NameLen) < len(data) {
		dirIndex.Name = string(data[offset : offset+int(dirIndex.NameLen)]) //?
		return offset + int(dirIndex.NameLen)
	} else {
		logger.MFTExtractorlogger.Warning(fmt.Sprintf("DirIdx nameLen %d exceeding available data %d", dirIndex.NameLen, len(data)))
		return len(data)
	}
}

func (dirIndex DirIndex) GetType() string {
	return DirTypes[dirIndex.Type]
}

func (dirIndex DirIndex) GetInfo() string {
	return fmt.Sprintf("transid  %d  type %s name %s", dirIndex.Transid, dirIndex.Name, dirIndex.GetType())
}
func (dirIndex DirIndex) ShowInfo() {
	fmt.Printf("%s \n", dirIndex.GetInfo())
}

func (extentData *ExtentData) Parse(data []byte) int {
	offset, _ := utils.Unmarshal(data, extentData)
	if extentData.Compression == 0 && extentData.Encryption == 0 &&
		ExtentTypes[extentData.Type] != "Inline Extent" {
		extentData.ExtentDataRem = new(ExtentDataRem)
		curOffset, _ := utils.Unmarshal(data[offset:], extentData.ExtentDataRem)
		offset += curOffset

	} else if extentData.Compression == 0 && extentData.Encryption == 0 &&
		ExtentTypes[extentData.Type] == "Inline Extent" {
		copy(extentData.InlineData, data[offset:])
		offset = len(data)
	}
	return offset
}

func (extentData ExtentData) ShowInfo() {
	fmt.Printf("%s \n", extentData.GetInfo())
}

func (extentData ExtentData) GetType() string {
	return ExtentTypes[extentData.Type]
}

func (extentData ExtentData) GetInfo() string {
	return fmt.Sprintf("%s %d", extentData.GetType(), extentData.LogicalDataLen)
}

func (extentItem *ExtentItem) Parse(data []byte) int {
	curOffset, _ := utils.Unmarshal(data, extentItem)
	extentItem.InlineRef = new(InlineRef)
	offset, _ := utils.Unmarshal(data[curOffset:], extentItem.InlineRef)
	curOffset += offset

	if ExtentItemTypes[uint8(extentItem.Flags)] == "BTRFS_EXTENT_FLAG_DATA" &&
		extentItem.InlineRef.Type == 0 { /// not shared
		extentItem.ExtentDataRef = new(ExtentDataRef)
		offset, _ = utils.Unmarshal(data[curOffset:], extentItem.ExtentDataRef)
	}
	return offset
}

func (extentItem ExtentItem) ShowInfo() {

}

func (extentItem ExtentItem) GetInfo() string {
	return ""
}

func (dirItem *DirItem) Parse(data []byte) int {
	offset, _ := utils.Unmarshal(data, dirItem)

	if offset+int(dirItem.NameLen) < len(data) { // needs check
		dirItem.Name = string(data[offset : offset+int(dirItem.NameLen)])
		if dirItem.DataLen > 1 {
			dirItem.XattValue = string(data[offset+int(dirItem.NameLen) : offset+int(dirItem.NameLen)+int(dirItem.DataLen)])
			return offset + int(dirItem.NameLen) + int(dirItem.DataLen)
		} else {
			return offset + int(dirItem.NameLen)
		}

	} else {
		logger.MFTExtractorlogger.Warning(fmt.Sprintf("Dir Item nameLen %d exceeding available data %d", dirItem.NameLen, len(data)))
		return len(data)
	}

}

func (dirItem DirItem) GetType() string {
	return DirTypes[dirItem.Type]
}

func (dirItem DirItem) ShowInfo() {
	fmt.Printf("%s \n", dirItem.GetInfo())
}

func (dirItem DirItem) GetInfo() string {
	return fmt.Sprintf("transid  %d  type %s name %s xattr vAL %s", dirItem.Transid, dirItem.GetType(), dirItem.Name, dirItem.XattValue)
}

func (inodeRef *InodeRef) Parse(data []byte) int {
	offset, _ := utils.Unmarshal(data, inodeRef)
	inodeRef.Name = string(data[offset : offset+int(inodeRef.Length)])
	return offset + int(inodeRef.Length)
}

func (inodeRef InodeRef) ShowInfo() {
	fmt.Printf("%s \n", inodeRef.GetInfo())
}

func (inodeRef InodeRef) GetInfo() string {
	return fmt.Sprintf("idx %d %s ", inodeRef.Index, inodeRef.Name)
}

func (item *Item) Parse(data []byte) int {
	item.Key = new(Key)
	item.Key.Parse(data)

	soffset, _ := utils.Unmarshal(data, item)
	return soffset

}

func (item Item) ShowInfo() {
	item.Key.ShowInfo()

}

func (key Key) IsDevItem() bool {
	return ItemTypes[int(key.ItemType)] == "DEV_ITEM"
}

func (key Key) IsDevExtentItem() bool {
	return ItemTypes[int(key.ItemType)] == "DEV_EXTENT"
}

func (item Item) GetType() string {
	return item.Key.GetType()
}

func (item Item) IsDevType() bool {
	return item.Key.IsDevItem()

}

func (item Item) IsChunkType() bool {
	return item.GetType() == "CHUNK_ITEM"
}

func (item Item) IsChecksumItem() bool {
	return item.GetType() == "EXTENT_CSUM"
}

func (item Item) IsExtentItem() bool {
	return item.GetType() == "EXTENT_ITEM"
}

func (item Item) IsExtentData() bool {
	return item.GetType() == "EXTENT_DATA"
}

func (item Item) IsRootRef() bool {
	return item.GetType() == "ROOT_REF"
}

func (item Item) IsRootBackRef() bool {
	return item.GetType() == "ROOT_BACKREF"
}

func (item Item) IsRootItem() bool {
	return item.GetType() == "ROOT_ITEM"
}

func (item Item) IsBlockGroupItem() bool {
	return item.GetType() == "BLOCK_GROUP_ITEM"
}

func (item Item) IsFSTree() bool {
	return ObjectTypes[int(item.Key.ObjectID)] == "FS_TREE"
}
func (item Item) IsDIRTree() bool {
	return ObjectTypes[int(item.Key.ObjectID)] == "ROOT_DIR_TREE"
}

func (item Item) IsExtentTree() bool {
	return ObjectTypes[int(item.Key.ObjectID)] == "EXTENT_TREE"
}

func (item Item) IsDEVTree() bool {
	return ObjectTypes[int(item.Key.ObjectID)] == "DEV_TREE"
}

func (item Item) IsInodeRef() bool {
	return item.GetType() == "INODE_REF"
}

func (item Item) IsDevItem() bool {
	return item.GetType() == "DEV_ITEM"
}

func (item Item) IsDirItem() bool {
	return item.GetType() == "DIR_ITEM"
}
func (item Item) IsInodeItem() bool {
	return item.GetType() == "INODE_ITEM"
}

func (item Item) IsXAttr() bool {
	return item.GetType() == "XATTR_ITEM"
}

func (item Item) IsDirIndex() bool {
	return item.GetType() == "DIR_INDEX"
}

func (item Item) IsMetadataItem() bool {
	return item.GetType() == "METADATA_ITEM"
}

func (item Item) IsDevStats() bool {
	return item.GetType() == "DEV_STATS"
}

func (leaf LeafNode) ShowInfo() {
	for idx, item := range leaf.Items {
		item.ShowInfo()
		if leaf.DataItems[idx] == nil {
			continue
		}
		leaf.DataItems[idx].ShowInfo()
		fmt.Printf("\n")
	}

}

func (item Item) Len() int {
	size := 0
	return utils.GetStructSize(item, size)
}

func (leaf *LeafNode) Parse(data []byte, physicalOffset int64) int {
	offset := 0

	for idx := range leaf.Items {
		offset += leaf.Items[idx].Parse(data[offset:])
	}

	for idx := range leaf.DataItems {
		item := leaf.Items[idx]

		if item.IsDevType() {
			leaf.DataItems[idx] = &DevItem{}
		} else if item.IsChunkType() {
			leaf.DataItems[idx] = &ChunkItem{}
		} else if item.IsInodeItem() {
			leaf.DataItems[idx] = &InodeItem{}
		} else if item.IsExtentItem() {
			leaf.DataItems[idx] = &ExtentItem{}
		} else if item.IsExtentData() {
			leaf.DataItems[idx] = &ExtentData{}
		} else if item.IsRootItem() {
			leaf.DataItems[idx] = &RootItem{}
		} else if item.IsInodeRef() {
			leaf.DataItems[idx] = &InodeRef{}
		} else if item.IsDirItem() {
			leaf.DataItems[idx] = &DirItem{}
		} else if item.IsDirIndex() {
			leaf.DataItems[idx] = &DirIndex{}
		} else if item.IsBlockGroupItem() {
			leaf.DataItems[idx] = &BlockGroupItem{}
		} else if item.IsMetadataItem() {
			leaf.DataItems[idx] = &ExtentItem{}
		} else if item.IsDevStats() {
			leaf.DataItems[idx] = &DevStatsItem{}
		} else if item.Key.IsDevExtentItem() {
			leaf.DataItems[idx] = &DevExtentItem{}
		} else if item.IsRootRef() {
			leaf.DataItems[idx] = &RootRef{}
		} else if item.IsRootBackRef() {
			leaf.DataItems[idx] = &RootBackRef{}
		} else if item.IsChecksumItem() {
			leaf.DataItems[idx] = &ChecksumItem{}
		} else if item.IsXAttr() {
			leaf.DataItems[idx] = &DirItem{}
		} else {
			logger.MFTExtractorlogger.Warning(fmt.Sprintf("Leaf at %d pos %d inodeid %d  %s  item type? %x", physicalOffset+int64(item.DataOffset),
				idx,
				item.Key.ObjectID,
				item.GetType(), item.Key.ItemType))
			continue
		}

		leaf.DataItems[idx].Parse(data[item.DataOffset : item.DataOffset+item.DataLen])

		logger.MFTExtractorlogger.Info(fmt.Sprintf("Leaf at %d pos %d inodeId %d %s %s", physicalOffset+int64(item.DataOffset),
			idx,
			item.Key.ObjectID,
			item.GetType(),
			leaf.DataItems[idx].GetInfo()))

	}

	return offset
}

func (inodeItem *InodeItem) Parse(data []byte) int {
	offset, _ := utils.Unmarshal(data, inodeItem)
	return offset
}

func (inodeItem InodeItem) ShowInfo() {

	fmt.Printf("%s \n", inodeItem.GetInfo())
}

func (inodeItem InodeItem) GetInfo() string {
	return fmt.Sprintf("C %s O %s M %s A %s", inodeItem.CTime.ToTime(), inodeItem.OTime.ToTime(),
		inodeItem.MTime.ToTime(), inodeItem.ATime.ToTime())
}

func (inodeItem InodeItem) GetType() string {
	return ItemTypes[int(inodeItem.Flags)]
}

func (rootItem *RootItem) Parse(data []byte) int {
	inode := new(InodeItem)
	inodeOffset := inode.Parse(data)
	offset, _ := utils.Unmarshal(data[inodeOffset:], rootItem)
	rootItem.Inode = inode
	return offset + inodeOffset
}

func (rootItem RootItem) ShowInfo() {
	fmt.Printf("%s ", rootItem.GetInfo())
}

func (rootItem RootItem) GetInfo() string {

	return fmt.Sprintf("block offset %d C %s O %s M %s A %s", rootItem.Bytenr,
		rootItem.CTime.ToTime(), rootItem.OTime.ToTime(),
		rootItem.MTime.ToTime(), rootItem.ATime.ToTime())
}

func (chunkItem *ChunkItem) Parse(data []byte) int {
	startOffset, _ := utils.Unmarshal(data, chunkItem)
	chunkItem.Stripes = make(ChunkItemStripes, chunkItem.NofStripes)
	for idx := range chunkItem.Stripes {
		offset, _ := utils.Unmarshal(data[startOffset:], &chunkItem.Stripes[idx])
		startOffset += offset
	}
	return startOffset
}

func (chunkitem ChunkItem) ShowInfo() {
	fmt.Printf("%s \n", chunkitem.GetInfo())
}

func (chunkItem ChunkItem) GetInfo() string {
	msg := fmt.Sprintf("stripe length %d owner %d io_alignment %d io_width %d num_stripes %d sub_stripes %d",
		chunkItem.StripeLen, chunkItem.Owner, chunkItem.OptimalIOAlignment,
		chunkItem.OptimalIOWidth, chunkItem.NofStripes, chunkItem.NofSubStripes)

	for idx, itemstrip := range chunkItem.Stripes {
		msg += fmt.Sprintf("stripe %d ", idx)
		msg += itemstrip.GetInfo()
	}

	return msg
}

func (devItem DevItem) ShowInfo() {

}

func (devItem DevItem) GetInfo() string {
	return ""
}

func (devItem *DevItem) Parse(data []byte) int {
	offset, _ := utils.Unmarshal(data, devItem)
	return offset
}

func (itemstrip ChunkItemStripe) ShowInfo() {
	itemstrip.GetInfo()
}

func (itemstrip ChunkItemStripe) GetInfo() string {
	return fmt.Sprintf("DevID %d DevUUID %s offset %d ", itemstrip.DeviceID, utils.StringifyGUID(itemstrip.DeviceUUID[:]), itemstrip.Offset)
}
