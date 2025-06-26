package leafnode

import (
	"fmt"

	"github.com/aarsakian/FileSystemForensics/FS/BTRFS/attributes"
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
	Key        *attributes.Key
	DataOffset uint32 //relative to the end of the header
	DataLen    uint32
}

// used when no compression, encryption other encoding is used non inline
type ExtentDataRem struct {
	LogicaAddress uint64 //logical address of extent
	Size          uint64
	Offset        uint64
	LogicalBytes  uint64
}

func (item *Item) Parse(data []byte) int {
	item.Key = new(attributes.Key)
	item.Key.Parse(data)

	soffset, _ := utils.Unmarshal(data, item)
	return soffset

}

func (item Item) ShowInfo() {
	item.Key.ShowInfo()

}

func (item Item) GetInfo() string {

	return fmt.Sprintf("obj %d offs %d Type %s  %s", item.Key.ObjectID, item.Key.Offset,
		attributes.ItemTypes[int(item.Key.ItemType)],
		attributes.ObjectTypes[int(item.Key.ObjectID)])
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
	return attributes.ObjectTypes[int(item.Key.ObjectID)] == "FS_TREE"
}
func (item Item) IsDIRTree() bool {
	return attributes.ObjectTypes[int(item.Key.ObjectID)] == "ROOT_DIR_TREE"
}

func (item Item) IsExtentTree() bool {
	return attributes.ObjectTypes[int(item.Key.ObjectID)] == "EXTENT_TREE"
}

func (item Item) IsDEVTree() bool {
	return attributes.ObjectTypes[int(item.Key.ObjectID)] == "DEV_TREE"
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
			leaf.DataItems[idx] = &attributes.DevItem{}
		} else if item.IsChunkType() {
			leaf.DataItems[idx] = &attributes.ChunkItem{}
		} else if item.IsInodeItem() {
			leaf.DataItems[idx] = &attributes.InodeItem{}
		} else if item.IsExtentItem() {
			leaf.DataItems[idx] = &attributes.ExtentItem{}
		} else if item.IsExtentData() {
			leaf.DataItems[idx] = &attributes.ExtentData{}
		} else if item.IsRootItem() {
			leaf.DataItems[idx] = &attributes.RootItem{}
		} else if item.IsInodeRef() {
			leaf.DataItems[idx] = &attributes.InodeRef{}
		} else if item.IsDirItem() {
			leaf.DataItems[idx] = &attributes.DirItem{}
		} else if item.IsDirIndex() {
			leaf.DataItems[idx] = &attributes.DirIndex{}
		} else if item.IsBlockGroupItem() {
			leaf.DataItems[idx] = &attributes.BlockGroupItem{}
		} else if item.IsMetadataItem() {
			leaf.DataItems[idx] = &attributes.ExtentItem{}
		} else if item.IsDevStats() {
			leaf.DataItems[idx] = &attributes.DevStatsItem{}
		} else if item.Key.IsDevExtentItem() {
			leaf.DataItems[idx] = &attributes.DevExtentItem{}
		} else if item.IsRootRef() {
			leaf.DataItems[idx] = &attributes.RootRef{}
		} else if item.IsRootBackRef() {
			leaf.DataItems[idx] = &attributes.RootBackRef{}
		} else if item.IsChecksumItem() {
			leaf.DataItems[idx] = &attributes.CsumItem{}
		} else if item.IsXAttr() {
			leaf.DataItems[idx] = &attributes.DirItem{}
		} else {
			logger.FSLogger.Warning(fmt.Sprintf("Leaf at %d pos %d inodeid %d  %s  item type? %x", physicalOffset+int64(item.DataOffset),
				idx,
				item.Key.ObjectID,
				item.GetType(), item.Key.ItemType))
			continue
		}

		leaf.DataItems[idx].Parse(data[item.DataOffset : item.DataOffset+item.DataLen])

		logger.FSLogger.Info(fmt.Sprintf("Leaf at %d pos %d inodeId %d %s %s", physicalOffset+int64(item.DataOffset),
			idx,
			item.Key.ObjectID,
			item.GetType(),
			leaf.DataItems[idx].GetInfo()))

	}

	return offset
}
