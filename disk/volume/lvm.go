package volume

import (
	"encoding/json"
	"errors"
	"fmt"

	metadata "github.com/aarsakian/FileSystemForensics/FS"
	"github.com/aarsakian/FileSystemForensics/logger"
	"github.com/aarsakian/FileSystemForensics/readers"
	"github.com/aarsakian/FileSystemForensics/utils"
)

type LVM2 struct {
	Header            *PhysicalVolLabel
	ConfigurationInfo string
	btrfs             *BTRFS
}

type PhysicalVolLabel struct {
	PhysicalVolLabelHeader *PhysicalVolLabelHeader
	PhysicalVolHeader      *PhysicalVolHeader
	MetadataAreaHeader     *MetadataAreaHeader
}

// 32bytes
type PhysicalVolLabelHeader struct {
	Signature     [8]byte
	SectorNum     uint64
	Chksum        [4]byte //20offst ot end
	HeaderSize    uint32
	IndicatorType [8]byte //8bytes
}

// 40+
type PhysicalVolHeader struct {
	UUID                [32]byte
	VolSize             uint64
	DataAreaDescriptors []DataAreaDescriptor
}

type DataAreaDescriptor struct {
	OffsetB int64 //from volume
	LenB    uint64
}

type MetadataAreaHeader struct {
	Chksum                 [4]byte
	Signature              [16]byte //LVM2
	Version                uint32
	Offset                 int64
	Size                   int64
	RawLocationDescriptors []RawLocationDescriptor
}

type RawLocationDescriptor struct {
	Offset int64
	Len    uint64
	Chksum [4]byte
	Flags  uint64
}

func (lvm2 *LVM2) ProcessHeader(hD readers.DiskReader, physicalOffsetB int64) error {
	data := hD.ReadFile(physicalOffsetB, 4096)
	lvm2.Parse(data)
	if !lvm2.HasValidSignature() {
		msg := "no lvm2 found"
		logger.FSLogger.Error(msg)
		fmt.Printf("%s \n", msg)
		return errors.New(msg)
	}
	data = hD.ReadFile(physicalOffsetB+4096, 512)
	lvm2.ParseMetaHeader(data)
	lvm2.ConfigurationInfo = string(hD.ReadFile(int64(physicalOffsetB)+4096+int64(lvm2.Header.MetadataAreaHeader.RawLocationDescriptors[0].Offset),
		int(lvm2.Header.MetadataAreaHeader.RawLocationDescriptors[0].Len)))
	return nil
}

func (lvm2 *LVM2) Process(hD readers.DiskReader, physicalOffsetB int64, SelectedEntries []int,
	fromEntry int, toEntry int) {
	btrfs := new(BTRFS)

	data := hD.ReadFile(physicalOffsetB+lvm2.Header.PhysicalVolHeader.DataAreaDescriptors[0].OffsetB+OFFSET_TO_SUPERBLOCK,
		SUPERBLOCKSIZE)
	err := btrfs.ParseSuperblock(data)
	if err != nil {
		return
	}
	btrfs.Process(hD, physicalOffsetB+lvm2.Header.PhysicalVolHeader.DataAreaDescriptors[0].OffsetB,
		SelectedEntries, fromEntry, toEntry)
	lvm2.btrfs = btrfs
}

func (lvm2 LVM2) HasValidSignature() bool {
	return string(lvm2.Header.PhysicalVolLabelHeader.Signature[:]) == "LABELONE"
}

// this will change
func (lvm2 LVM2) GetFS() []metadata.Record {
	var records []metadata.Record
	for _, tree := range lvm2.btrfs.FsTreeMap {
		for _, record := range tree.FilesDirsMap {
			temp := record
			records = append(records, metadata.BTRFSRecord{&temp})
		}
	}
	return records
}
func (lvm2 *LVM2) Parse(data []byte) {
	header := new(PhysicalVolLabel)
	header.Parse(data[512:])
	lvm2.Header = header

}

func (lvm2 LVM2) GetLogicalToPhysicalMap() map[uint64]metadata.Chunk {
	return lvm2.btrfs.GetLogicalToPhysicalMap()
}

func (lvm2 *LVM2) ParseMetaHeader(data []byte) {
	metadataAreaHeader := new(MetadataAreaHeader)
	offset, _ := utils.Unmarshal(data, metadataAreaHeader)
	idx := 0
	metadataAreaHeader.RawLocationDescriptors = make([]RawLocationDescriptor, 4)

	for idx < 4 {
		rawLocationDescriptorSize := utils.GetStructSize(metadataAreaHeader.RawLocationDescriptors[idx], 0)
		utils.Unmarshal(data[offset+idx*rawLocationDescriptorSize:offset+(idx+1)*rawLocationDescriptorSize],
			&metadataAreaHeader.RawLocationDescriptors[idx])
		idx++

	}

	lvm2.Header.MetadataAreaHeader = metadataAreaHeader

}

func (physicalVolHeader *PhysicalVolHeader) Parse(data []byte) {

	offset, _ := utils.Unmarshal(data, physicalVolHeader)

	dataDescriptor := new(DataAreaDescriptor)
	currOffset, _ := utils.Unmarshal(data[offset:], dataDescriptor)

	for dataDescriptor.OffsetB != 0 {
		physicalVolHeader.DataAreaDescriptors = append(physicalVolHeader.DataAreaDescriptors, *dataDescriptor)
		offset, _ := utils.Unmarshal(data[offset+currOffset:], dataDescriptor)

		currOffset += offset

	}

}

func (header *PhysicalVolLabel) Parse(data []byte) {
	phyVolLabelHeader := new(PhysicalVolLabelHeader)
	utils.Unmarshal(data, phyVolLabelHeader)
	physicalVolHeader := new(PhysicalVolHeader)
	physicalVolHeader.Parse(data[phyVolLabelHeader.HeaderSize:])

	header.PhysicalVolLabelHeader = phyVolLabelHeader
	header.PhysicalVolHeader = physicalVolHeader

}

func (lvm2 LVM2) GetBytesPerSector() uint64 {
	return 512
}

func (lvm2 LVM2) GetSectorsPerCluster() int {
	return lvm2.btrfs.GetSectorsPerCluster()
}

func (lvm2 LVM2) GetSignature() string {
	return string(lvm2.Header.PhysicalVolLabelHeader.Signature[:])
}

func (lvm2 LVM2) GetUnallocatedClusters() []int {
	return []int{}
}

func (lvm2 LVM2) GetInfo() string {
	prettyJson, err := json.MarshalIndent(lvm2.ConfigurationInfo, "", " ")
	if err != nil {
		return ""
	}
	return string(prettyJson)

}
