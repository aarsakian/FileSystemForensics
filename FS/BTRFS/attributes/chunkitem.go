package attributes

import (
	"fmt"

	"github.com/aarsakian/FileSystemForensics/utils"
)

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

type ChunkItemStripes []ChunkItemStripe

type ChunkItemStripe struct {
	DeviceID   uint64
	Offset     uint64
	DeviceUUID [16]byte
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

func (itemstrip ChunkItemStripe) ShowInfo() {
	itemstrip.GetInfo()
}

func (itemstrip ChunkItemStripe) GetInfo() string {
	return fmt.Sprintf("DevID %d DevUUID %s offset %d ", itemstrip.DeviceID, utils.StringifyGUID(itemstrip.DeviceUUID[:]), itemstrip.Offset)
}
