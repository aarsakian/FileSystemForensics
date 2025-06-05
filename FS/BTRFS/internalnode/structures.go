package internalnode

import (
	"reflect"

	"github.com/aarsakian/FileSystemForensics/utils"
)

type Key struct {
	ObjectID uint64
	ItemType uint8
	Offset   uint64
}

type InternalNode struct {
	Items []BlockPointer
}

type BlockPointer struct { //33B
	Key                     *Key
	LogicalAddressRefHeader uint64
	Generation              uint64
}

func (blockPointer BlockPointer) GetSize() int {
	size := 0
	return utils.GetStructSize(blockPointer, size)
}

func (internalNode *InternalNode) Parse(data []byte, physicalOffset int64) int {
	startOffset := 0

	for idx := range internalNode.Items {

		startOffset += internalNode.Items[idx].Parse(data[startOffset:])

	}

	return startOffset
}

func (blockPointer *BlockPointer) Parse(data []byte) int {
	key := new(Key)
	if len(data) < int(reflect.TypeOf(reflect.ValueOf(key).Elem()).Size()) {
		return 0
	}
	utils.Unmarshal(data, key)

	if len(data) < int(reflect.TypeOf(reflect.ValueOf(blockPointer).Elem()).Size()) {
		return 0
	}
	offset, _ := utils.Unmarshal(data, blockPointer)
	blockPointer.Key = key
	return offset

}
