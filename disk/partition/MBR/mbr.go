package MBR

import (
	"errors"
	"fmt"

	volume "github.com/aarsakian/FileSystemForensics/disk/volume"
	"github.com/aarsakian/FileSystemForensics/logger"
	"github.com/aarsakian/FileSystemForensics/readers"
	"github.com/aarsakian/FileSystemForensics/utils"
)

var PartitionTypes = map[uint8]string{
	0x07: "HPFS/NTFS/exFAT",
	0x0c: "W95 FAT32 (LBA)",
	0x0f: "Extended",
	0x27: "Hidden NTFS Win"}

type MBR struct {
	BootCode           [446]byte //0-445
	Partitions         []Partition
	ExtendedPartitions []ExtendedPartition
	Signature          []byte //510-511
}

type ExtendedPartition struct {
	Partition   *Partition
	TableOffset int
}
type Partition struct {
	Flag     uint8
	StartCHS [3]byte
	Type     uint8
	EndCHS   [3]byte
	StartLBA uint32
	Size     uint32 //sectors
	Volume   volume.Volume
}

func (partition Partition) GetOffset() uint64 {
	return uint64(partition.StartLBA)
}

func (partition Partition) GetPartitionType() string {
	return PartitionTypes[partition.Type]
}

func (partition *Partition) LocateVolume(hD readers.DiskReader) {

	if partition.Type == 0x07 || partition.Type == 0x17 {
		partitionOffetB := uint64(partition.GetOffset() * 512)
		data := hD.ReadFile(int64(partitionOffetB), 512)
		ntfs := new(volume.NTFS)
		ntfs.AddVolume(data)

		if ntfs.HasValidSignature() {
			partition.Volume = ntfs
		}
	} else if partition.Type == 0x83 { //linux native
		btrfs := new(volume.BTRFS)

		partitionOffsetB := int64(partition.GetOffset() * 512)
		data := hD.ReadFile(partitionOffsetB+volume.OFFSET_TO_SUPERBLOCK, volume.SUPERBLOCKSIZE)
		msg := "Reading superblock at offset %d"
		fmt.Printf(msg+"\n", partitionOffsetB)
		logger.FSLogger.Info(fmt.Sprintf(msg, partitionOffsetB))

		err := btrfs.ParseSuperblock(data)
		if err == nil && btrfs.HasValidSignature() {
			partition.Volume = btrfs
		}
	}

}

func (extPartition ExtendedPartition) GetOffset() uint64 {
	return uint64(extPartition.Partition.StartLBA) + uint64(extPartition.TableOffset)
}

func (extPartition *ExtendedPartition) LocateVolume(hD readers.DiskReader) {
	extPartition.Partition.LocateVolume(hD)
}

func (mbr MBR) IsProtective() bool {
	return mbr.Partitions[0].Type == 0xEE // 1st partition flag
}

func (mbr MBR) GetPartition(partitionNum int) Partition {
	return mbr.Partitions[partitionNum]
}

func LocatePartitions(data []byte) []Partition {
	pos := 0
	var partitions []Partition
	for pos < len(data) {
		var partition *Partition = new(Partition) //explicit is better
		utils.Unmarshal(data[pos:pos+16], partition)
		if partition.Type == 0x00 {
			break
		}
		partitions = append(partitions, *partition)
		pos += 16
	}

	return partitions
}

func (mbr *MBR) PopulatePseudoMBR(voltype string) {
	partition := new(Partition)

	utils.Unmarshal(make([]byte, 16), partition)
	if voltype == "NTFS" {
		partition.Type = 0x07
	}
	mbr.Partitions = []Partition{*partition}
}

func (mbr *MBR) DiscoverExtendedPartitions(buffer []byte, offset int) {
	var extPartitions []ExtendedPartition
	partitions := LocatePartitions(buffer[446:510])
	for idx := range partitions {
		extPartitions = append(extPartitions, ExtendedPartition{Partition: &partitions[idx], TableOffset: offset})
	}
	mbr.ExtendedPartitions = extPartitions
}

func (mbr *MBR) Parse(buffer []byte) {

	utils.Unmarshal(buffer, mbr)
	mbr.Partitions = LocatePartitions(buffer[446:510])
	mbr.Signature = buffer[510:]

}

func (mbr MBR) GetExtendedPartitionOffset() (int, error) {
	for _, partition := range mbr.Partitions {
		if partition.Type == 0x0f {
			return int(partition.GetOffset()), nil
		}
	}
	return -1, errors.New("extended partition not found")
}

func (mbr *MBR) UpdateExtendedPartitionsOffsets(extendedTableSectorOffset uint32) {
	for idx := range mbr.Partitions {
		if mbr.Partitions[idx].Type != 0x0f {
			continue
		}
		mbr.Partitions[idx].StartLBA += extendedTableSectorOffset
	}
}

func (partition Partition) GetVolInfo() string {
	return ""
}

func (partiton Partition) GetVolume() volume.Volume {
	return partiton.Volume
}

func (extPartition ExtendedPartition) GetVolume() volume.Volume {
	return extPartition.Partition.Volume
}

func (partition Partition) GetInfo() string {
	return fmt.Sprintf(" %s at %d size %d sectors", partition.GetPartitionType(), partition.GetOffset(), partition.Size)

}

func (extPartition ExtendedPartition) GetInfo() string {

	return fmt.Sprintf("\textended partition  %s at %d size %d sectors",
		extPartition.Partition.GetPartitionType(), extPartition.Partition.GetOffset(), extPartition.Partition.Size)
}

func (extpartition ExtendedPartition) GetVolInfo() string {
	return ""
}
