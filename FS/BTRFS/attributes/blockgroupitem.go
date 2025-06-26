package attributes

import "github.com/aarsakian/FileSystemForensics/utils"

type BlockGroupItem struct {
	Used          uint64
	ChunkObjectID uint64
	Flags         uint64
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
