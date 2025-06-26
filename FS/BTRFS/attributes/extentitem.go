package attributes

import "github.com/aarsakian/FileSystemForensics/utils"

type ExtentItem struct {
	RefCount         uint64
	Generation       uint64
	Flags            uint64 //contain data or metadata
	InlineRef        *InlineRef
	ExtentDataRef    *ExtentDataRef
	ExtentDataRefKey *ExtentDataRefKey
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

type InlineRef struct {
	Type   uint8 //extent is shared or not
	Offset uint64
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
