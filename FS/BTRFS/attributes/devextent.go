package attributes

import "github.com/aarsakian/FileSystemForensics/utils"

type DevExtentItem struct {
	ChunkTree     uint64
	ChunkObjectID uint64
	ChunkOffset   uint64
	Length        uint64
	UUID          [16]byte
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
