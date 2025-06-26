package attributes

import (
	"fmt"

	"github.com/aarsakian/FileSystemForensics/utils"
)

type Key struct {
	ObjectID uint64
	ItemType uint8
	Offset   uint64
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

func (key Key) IsDevItem() bool {
	return ItemTypes[int(key.ItemType)] == "DEV_ITEM"
}

func (key Key) IsDevExtentItem() bool {
	return ItemTypes[int(key.ItemType)] == "DEV_EXTENT"
}
