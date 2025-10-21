package attributes

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/aarsakian/FileSystemForensics/logger"
	"github.com/aarsakian/FileSystemForensics/readers"
	"github.com/aarsakian/FileSystemForensics/utils"
)

var AttrTypes = map[uint32]string{
	0x00000010: "Standard Information", 0x00000020: "Attribute List",
	0x00000030: "FileName", 0x00000040: "Object ID",
	0x00000050: "Security Descriptor", 0x00000060: "Volume Name",
	0x00000070: "Volume Information", 0x00000080: "DATA",
	0x00000090: "Index Root", 0x000000a0: "Index Allocation",
	0x000000b0: "BitMap", 0x000000c0: "Reparse Point",
	0x000000d0: "Extended Attribute Information",
	0x000000e0: "Extended Attribute", 0x000000f0: "Extended Attribute Information",
	0x00000100: "Logged Utility Stream", 0x00000110: "$TXF_Data",
	0xffffffff: "Last",
}

type AttributeHeader struct {
	Type                 [4]byte //        0-3                              type of attribute e.g. $DATA
	AttrLen              uint16  //4-8             length of attribute??? practice shown 4-6
	Uknown               [2]byte
	NoNResident          uint8  //8
	Nlen                 uint8  //9
	NameOff              uint16 //name offset 10-12          relative to the start of attribute
	Flags                uint16 //12-14           //compressed,
	ID                   uint16 //14-16 type of attribute
	ATRrecordResident    *ATRrecordResident
	ATRrecordNoNResident *ATRrecordNoNResident
}

type ATRrecordResident struct {
	ContentSize   uint32 //16-20 size of Resident attribute
	OffsetContent uint16 //20-22 offset to content            soff+ssize=len
	IdxFlags      uint16
	Name          string
}

type ATRrecordNoNResident struct {
	StartVcn          uint64   //16-24
	LastVcn           uint64   //24-32
	RunOff            uint16   //32-34     offset to the start of the attribute
	Compusize         uint16   //34-36
	F1                uint32   //36-40
	AllocatedLength   uint64   //40-48
	ActualLength      uint64   //48-56
	InitLength        uint64   //56-64
	RunList           *RunList //holds a linked list of runs
	RunListTotalLenCl uint64   // total length of runlist
}

type RunList struct {
	Offset int64
	Length uint64
	Next   *RunList
}

func (atrRecordResident *ATRrecordResident) Parse(data []byte) {
	utils.Unmarshal(data[:8], atrRecordResident)
}

func (atrRecordNoNResident ATRrecordNoNResident) GetContent(hD readers.DiskReader, partitionOffsetB int64, clusterSizeB int,
	buf *bytes.Buffer) error {

	if atrRecordNoNResident.RunList == nil {
		msg := "non resident attribute has no runlists"
		logger.FSLogger.Warning(msg)
		return errors.New(msg)
	}

	runlist := atrRecordNoNResident.RunList

	offset := int64(0)
	for runlist != nil {

		offset += int64(runlist.Offset)
		data, _ := hD.ReadFile(partitionOffsetB+offset*int64(clusterSizeB),
			int(runlist.Length)*clusterSizeB)
		buf.Write(data)

		if runlist.Next == nil {
			break
		}

		runlist = runlist.Next
	}

	return nil
}

func (attrHeader AttributeHeader) GetType() string {

	attrType, ok := AttrTypes[utils.ToUint32(attrHeader.Type[:])]
	if ok {
		return attrType
	} else {
		return fmt.Sprintf("%s ", attrHeader.Type)
	}
}

func (attrHeader AttributeHeader) IsLast() bool {
	return attrHeader.GetType() == "Last"
}

func (attrHeader AttributeHeader) IsFileName() bool {
	return attrHeader.GetType() == "FileName"
}

func (attrHeader AttributeHeader) IsData() bool {
	return attrHeader.GetType() == "DATA"
}

func (attrHeader AttributeHeader) IsReparse() bool {
	return attrHeader.GetType() == "Reparse Point"
}

func (attrHeader AttributeHeader) IsObject() bool {
	return attrHeader.GetType() == "Object ID"
}

func (attrHeader AttributeHeader) IsAttrList() bool {
	return attrHeader.GetType() == "Attribute List"
}

func (attrHeader AttributeHeader) IsBitmap() bool {
	return attrHeader.GetType() == "BitMap"
}

func (attrHeader AttributeHeader) IsVolumeName() bool {
	return attrHeader.GetType() == "Volume Name"
}

func (attrHeader AttributeHeader) IsIndexAllocation() bool {
	return attrHeader.GetType() == "Index Allocation"
}

func (attrHeader AttributeHeader) IsVolumeInfo() bool {
	return attrHeader.GetType() == "Volume Information"
}

func (attrHeader AttributeHeader) IsIndexRoot() bool {
	return attrHeader.GetType() == "Index Root"
}

func (attrHeader AttributeHeader) IsStdInfo() bool {
	return attrHeader.GetType() == "Standard Information"
}

func (attrHeader AttributeHeader) IsLoggedUtility() bool {
	return attrHeader.GetType() == "Logged Utility Stream"
}

func (attrHeader AttributeHeader) IsExtendedAttribute() bool {
	return attrHeader.GetType() == "Extended Attribute"
}

func (attrHeader AttributeHeader) IsExtendedInformationAttribute() bool {
	return attrHeader.GetType() == "Extended Attribute Information"
}

func (attrHeader AttributeHeader) IsNoNResident() bool {
	return attrHeader.NoNResident == 1
}

func (attrHeader AttributeHeader) GetName() string {
	if !attrHeader.IsNoNResident() {
		return attrHeader.ATRrecordResident.Name
	} else {
		return ""
	}
}

func (prevRunlist *RunList) Process(runlists []byte) uint64 {
	clusterPtr := uint64(0)
	length := uint64(0)
	for clusterPtr < uint64(len(runlists)) { // length of bytes of runlist
		ClusterOffsB := uint64(runlists[clusterPtr] & 0xf0 >> 4)
		ClusterLenB := uint64(runlists[clusterPtr] & 0x0f)

		if ClusterLenB != 0 { //sparse or compressed attribute offset is zero
			clustersLen := utils.ReadEndianUInt(runlists[clusterPtr+1 : clusterPtr+
				ClusterLenB+1])

			clustersOff := utils.ReadEndianInt(runlists[clusterPtr+1+
				ClusterLenB : clusterPtr+ClusterLenB+ClusterOffsB+1])

			runlist := RunList{Offset: clustersOff, Length: clustersLen}

			length += clustersLen

			if clusterPtr == 0 {
				*prevRunlist = runlist
			} else {
				prevRunlist.Next = &runlist
				prevRunlist = &runlist
			}

			//		prevRunlist = runlist
			clusterPtr += ClusterLenB + ClusterOffsB + 1

		} else {
			break
		}
	}
	return length
}
