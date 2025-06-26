package attributes

import (
	"fmt"

	"github.com/aarsakian/FileSystemForensics/utils"
)

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

func (inodeItem *InodeItem) Parse(data []byte) int {
	offset, _ := utils.Unmarshal(data, inodeItem)
	return offset
}

func (inodeItem InodeItem) ShowInfo() {

	fmt.Printf("%s \n", inodeItem.GetInfo())
}

func (inodeItem InodeItem) GetInfo() string {
	return inodeItem.GetType()
}

func (inodeItem InodeItem) GetTimestamps() string {
	return fmt.Sprintf("C %s O %s M %s A %s", inodeItem.CTime.ToTime(), inodeItem.OTime.ToTime(),
		inodeItem.MTime.ToTime(), inodeItem.ATime.ToTime())
}

func (inodeItem InodeItem) GetType() string {
	return ItemTypes[int(inodeItem.Flags)]
}

/*SizeB: int(dataItem.StSize), Nlink: int(dataItem.StNlink),
Uid: int(dataItem.StUid), Gid: int(dataItem.StGid), ATime: dataItem.ATime.ToTime(),
MTime: dataItem.MTime.ToTime(),
CTime: dataItem.CTime.ToTime(), OTime: dataItem.OTime.ToTime(), Type: dataItem.GetType()}*/
