package gpt

import (
	"fmt"

	mdraid "github.com/aarsakian/FileSystemForensics/disk/raid"
	"github.com/aarsakian/FileSystemForensics/disk/volume"
	lvmlib "github.com/aarsakian/FileSystemForensics/disk/volume"
	"github.com/aarsakian/FileSystemForensics/readers"
	"github.com/aarsakian/FileSystemForensics/utils"
)

var PartitionTypeGuids = map[string]string{
	"ebd0a0a2-b9e5-4433-87c0-68b6b72699c7": "Basic Data Partition",
	"c12a7328-f81f-11d2-ba4b-00a0c93ec93b": "EFI System Partition",
	"de94bba4-06d1-4d40-a16a-bfd50179d6ac": "Windows Recovery Environment",
	"e3c9e316-0b5c-4db8-817d-f92df00215ae": "Microsoft Reserved Partition",
	"a19d880f-05fc-4d3b-a006-743f0f84911e": "Linux RAID",
}

type GPT struct {
	Header     *GPTHeader
	Partitions []Partition
}
type GPTHeader struct {
	StartSignature     [8]byte
	Revision           [4]byte
	HeaderSize         uint32
	HeaderCRC          uint32
	Reserved           [4]byte
	CurrentLBA         uint64 //location of header
	BackupLBA          uint64
	FirstUsableLBA     uint64
	LastUsableLBA      uint64
	DiskGUID           [16]byte
	PartitionsStartLBA uint64 // usually LBA 2
	NofPartitions      uint32
	PartitionSize      uint32
	PartionArrayCRC    uint32
	Reserved2          [418]byte
	EndSignature       [2]byte //510-511
}

type Partition struct {
	PartitionTypeGUID [16]byte
	PartitionGUID     [16]byte
	StartLBA          uint64
	EndLBA            uint64
	Atttributes       [8]byte
	Name              string
	Volume            volume.Volume
	Raid              *mdraid.Superblock
}

func (partition Partition) GetPartitionType() string {
	guid := utils.StringifyGUID(partition.PartitionTypeGUID[:])
	partitionType, ok := PartitionTypeGuids[guid]
	if ok {
		return partitionType
	} else {
		return guid
	}
}

func (partition Partition) GetUniquePartitionType() string {
	return utils.StringifyGUID(partition.PartitionGUID[:])
}

func (partition Partition) IdentifyType() string {
	return PartitionTypeGuids[partition.GetPartitionType()]
}

func (gpt *GPT) ParseHeader(buffer []byte) {

	var header GPTHeader
	utils.Unmarshal(buffer, &header)
	gpt.Header = &header

}

func (gpt GPT) GetPartitionArraySize() uint32 {
	return gpt.Header.PartitionSize * gpt.Header.NofPartitions
}

func (gpt *GPT) ParsePartitions(data []byte) {

	partitions := make([]Partition, gpt.Header.NofPartitions)

	for idx := 0; idx < len(partitions); idx++ {
		var partition Partition
		utils.Unmarshal(data[idx*int(gpt.Header.PartitionSize):(idx+1)*int(gpt.Header.PartitionSize)], &partition)
		partitions[idx] = partition
	}
	gpt.Partitions = partitions
}

func (gpt GPT) GetPartition(partitionNum int) Partition {
	return gpt.Partitions[partitionNum]
}

func (partition Partition) GetOffset() uint64 {
	offset := uint64(0)
	if partition.Raid != nil {
		offset = partition.Raid.DataOffset
	}

	return partition.StartLBA + offset
}

func (partition Partition) GetVolInfo() string {
	if partition.Volume != nil {
		return partition.Volume.GetInfo()
	} else {
		return ""
	}

}

func (partition *Partition) LocateVolume(hD readers.DiskReader) {
	partitionOffetB := uint64(partition.GetOffset() * 512)

	if partition.GetPartitionType() == "Basic Data Partition" {

		data := hD.ReadFile(int64(partitionOffetB), 512)

		ntfs := new(volume.NTFS)
		ntfs.AddVolume(data)
		partition.Volume = ntfs
	} else if partition.GetPartitionType() == "Linux RAID" {
		data := hD.ReadFile(int64(partitionOffetB+8*512), 512)       //8 sectors after superblock
		if utils.Hexify(utils.Bytereverse(data[:4])) == "a92b4efc" { //valid ?
			superblock := new(mdraid.Superblock)
			utils.Unmarshal(data, superblock)
			partition.Raid = superblock

			lvm2 := new(lvmlib.LVM2)
			lvm2.ProcessHeader(hD, int64(partitionOffetB+superblock.DataOffset*512))
			partition.Volume = lvm2
		}

	} else {
		partition.Volume = nil
	}

}

func (partition Partition) GetVolume() volume.Volume {
	return partition.Volume
}

func (partition Partition) GetInfo() string {
	return fmt.Sprintf("%s  %s at %d", partition.GetUniquePartitionType(),
		partition.GetPartitionType(), partition.GetOffset())
}
