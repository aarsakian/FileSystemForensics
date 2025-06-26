package attributes

import (
	"fmt"

	"github.com/aarsakian/FileSystemForensics/utils"
)

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
