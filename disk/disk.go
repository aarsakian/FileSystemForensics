package disk

import (
	"errors"
	"fmt"
	"sync"

	"github.com/aarsakian/FileSystemForensics/FS/NTFS/MFT"
	gptLib "github.com/aarsakian/FileSystemForensics/disk/partition/GPT"
	mbrLib "github.com/aarsakian/FileSystemForensics/disk/partition/MBR"
	"github.com/aarsakian/FileSystemForensics/disk/volume"
	"github.com/aarsakian/FileSystemForensics/img"
	"github.com/aarsakian/FileSystemForensics/logger"
	"github.com/aarsakian/FileSystemForensics/utils"
)

var ErrNTFSVol = errors.New("NTFS volume discovered instead of MBR")

type Disk struct {
	MBR        *mbrLib.MBR
	GPT        *gptLib.GPT
	Handler    img.DiskReader
	Partitions []Partition
}

func (disk *Disk) Initialize(evidencefile string, physicaldrive int, vmdkfile string) {
	var hD img.DiskReader
	if evidencefile != "" {

		hD = img.GetHandler(evidencefile, "ewf")

	} else if physicaldrive != -1 {

		hD = img.GetHandler(fmt.Sprintf("\\\\.\\PHYSICALDRIVE%d", physicaldrive), "physicalDrive")

	} else {

		hD = img.GetHandler(vmdkfile, "vmdk")

	}
	disk.Handler = hD
}

func (disk *Disk) Process(partitionNum int, MFTentries []int, fromMFTEntry int, toMFTEntry int) map[int]MFT.Records {

	err := disk.DiscoverPartitions(partitionNum)
	if errors.Is(err, ErrNTFSVol) {
		msg := "No MBR discovered, instead NTFS volume found at 1st sector"
		fmt.Printf("%s\n", msg)
		logger.MFTExtractorlogger.Warning(msg)

		disk.CreatePseudoMBR("NTFS")
	}
	disk.ProcessPartitions(partitionNum)

	disk.DiscoverFileSystems(MFTentries, fromMFTEntry, toMFTEntry)

	return disk.GetFileSystemMetadata(partitionNum)
}

func (disk Disk) Close() {
	disk.Handler.CloseHandler()
}

func (disk Disk) hasProtectiveMBR() bool {
	return disk.MBR.IsProtective()
}

type Partition interface {
	GetOffset() uint64
	LocateVolume(img.DiskReader)
	GetVolume() volume.Volume
	GetInfo() string
	GetVolInfo() string
}

func (disk *Disk) DiscoverFileSystems(MFTentries []int, fromMFTEntry int, toMFTEntry int) {
	for idx := range disk.Partitions {

		vol := disk.Partitions[idx].GetVolume()
		if vol == nil {
			continue
		}
		partitionOffsetB := int64(disk.Partitions[idx].GetOffset() *
			vol.GetBytesPerSector())

		vol.Process(disk.Handler, partitionOffsetB, MFTentries, fromMFTEntry, toMFTEntry)

	}
}

func (disk *Disk) populateMBR() error {
	var mbr mbrLib.MBR
	physicalOffset := int64(0)
	length := int(512) // MBR always at first sector

	data := disk.Handler.ReadFile(physicalOffset, length) // read 1st sector

	if string(data[3:7]) == "NTFS" {
		return ErrNTFSVol
	}

	mbr.Parse(data)
	offset, err := mbr.GetExtendedPartitionOffset()
	if err == nil {
		data := disk.Handler.ReadFile(physicalOffset+int64(offset)*512, length)
		mbr.DiscoverExtendedPartitions(data, offset)

	}
	disk.MBR = &mbr
	return nil
}

func (disk *Disk) populateGPT() {

	physicalOffset := int64(512) // gpt always starts at 512

	data := disk.Handler.ReadFile(physicalOffset, 512)

	var gpt gptLib.GPT
	gpt.ParseHeader(data)
	length := gpt.GetPartitionArraySize()

	data = disk.Handler.ReadFile(int64(gpt.Header.PartitionsStartLBA*512), int(length))

	gpt.ParsePartitions(data)

	disk.GPT = &gpt
}

func (disk *Disk) CreatePseudoMBR(voltype string) {
	var mbr mbrLib.MBR

	mbr.PopulatePseudoMBR(voltype)
	disk.MBR = &mbr
	for _, partition := range disk.MBR.Partitions {
		disk.Partitions = append(disk.Partitions, &partition)
	}

}

func (disk *Disk) DiscoverPartitions(partitionNum int) error {

	err := disk.populateMBR()
	if err != nil {
		return err
	}
	if disk.hasProtectiveMBR() {
		disk.populateGPT()
		for idx := range disk.GPT.Partitions {
			if partitionNum != -1 && partitionNum != idx {
				continue
			}
			disk.Partitions = append(disk.Partitions, &disk.GPT.Partitions[idx])

		}

	} else {
		for idx := range disk.MBR.Partitions {
			disk.Partitions = append(disk.Partitions, &disk.MBR.Partitions[idx])
		}
		for idx := range disk.MBR.ExtendedPartitions {
			disk.Partitions = append(disk.Partitions, &disk.MBR.ExtendedPartitions[idx])
		}
	}
	return nil
}

func (disk *Disk) ProcessPartitions(partitionNum int) {

	for idx := range disk.Partitions {
		if partitionNum != -1 && idx+1 != partitionNum {
			continue
		}

		disk.Partitions[idx].LocateVolume(disk.Handler)
		parttionOffset := disk.Partitions[idx].GetOffset()
		vol := disk.Partitions[idx].GetVolume()
		if vol == nil {
			msg := "No Known Volume at partition %d (Currently supported NTFS)."
			logger.MFTExtractorlogger.Error(fmt.Sprintf(msg, idx))
			continue //fs not found
		}
		msg := "Partition %d  %s at %d sector"
		fmt.Printf(msg+"\n", idx+1, vol.GetSignature(), parttionOffset)
		logger.MFTExtractorlogger.Info(fmt.Sprintf(msg, idx+1, vol.GetSignature(), parttionOffset))

	}

}

func (disk Disk) GetFileSystemMetadata(partitionNum int) map[int]MFT.Records {

	recordsPerPartition := map[int]MFT.Records{}
	for idx, partition := range disk.Partitions {
		if partitionNum != -1 && idx+1 != partitionNum {
			continue
		}
		vol := partition.GetVolume()
		if vol == nil {
			continue
		}
		//	recordsPerPartition[idx] = fs.GetMetadata()

	}
	return recordsPerPartition
}

func (disk Disk) AsyncWorker(wg *sync.WaitGroup, record MFT.Record, dataClusters chan<- []byte, partitionNum int) {
	defer wg.Done()
	partition := disk.Partitions[partitionNum]

	vol := partition.GetVolume()
	sectorsPerCluster := int(vol.GetSectorsPerCluster())
	bytesPerSector := int(vol.GetBytesPerSector())
	partitionOffsetB := int64(partition.GetOffset()) * int64(bytesPerSector)

	if record.IsFolder() {
		msg := fmt.Sprintf("Record %s Id %d is folder! No data to export.", record.GetFname(), record.Entry)
		logger.MFTExtractorlogger.Warning(msg)
		close(dataClusters)
		return
	}
	fmt.Printf("pulling data file %s Id %d\n", record.GetFname(), record.Entry)

	if len(record.LinkedRecords) == 0 {
		record.LocateDataAsync(disk.Handler, partitionOffsetB, sectorsPerCluster, bytesPerSector, dataClusters)
	} else { // attribute runlist

		for _, linkedRecord := range record.LinkedRecords {
			linkedRecord.LocateDataAsync(disk.Handler, partitionOffsetB, sectorsPerCluster, bytesPerSector, dataClusters)

		}
	}
	// use lsize to make sure that we cannot exceed the logical size

	close(dataClusters)

}
func (disk Disk) Worker(wg *sync.WaitGroup, records MFT.Records, results chan<- utils.AskedFile, partitionNum int) {
	defer wg.Done()
	partition := disk.Partitions[partitionNum]

	vol := partition.GetVolume()
	sectorsPerCluster := int(vol.GetSectorsPerCluster())
	bytesPerSector := int(vol.GetBytesPerSector())
	partitionOffsetB := int64(partition.GetOffset()) * int64(bytesPerSector)

	for _, record := range records {

		if record.IsFolder() {
			msg := fmt.Sprintf("Record %s Id %d is folder! No data to export.", record.GetFname(), record.Entry)
			logger.MFTExtractorlogger.Warning(msg)
			continue
		}
		fmt.Printf("pulling data file %s Id %d\n", record.GetFname(), record.Entry)

		if len(record.LinkedRecords) == 0 {
			record.LocateData(disk.Handler, partitionOffsetB, sectorsPerCluster, bytesPerSector, results)
		} else { // attribute runlist

			for _, linkedRecord := range record.LinkedRecords {
				linkedRecord.LocateData(disk.Handler, partitionOffsetB, sectorsPerCluster, bytesPerSector, results)

			}
		}
		// use lsize to make sure that we cannot exceed the logical size

	}
	close(results)

}

func (disk Disk) ShowVolumeInfo() {
	for _, partition := range disk.Partitions {
		offset := partition.GetOffset()
		//show only non zero partition entries
		if offset == 0 {
			continue
		}
		fmt.Printf("%s \n", partition.GetVolInfo())
	}
}

func (disk Disk) ListPartitions() {
	if disk.hasProtectiveMBR() {
		fmt.Printf("GPT:\n")
	} else {
		fmt.Printf("MBR:\n")
	}

	for _, partition := range disk.Partitions {
		offset := partition.GetOffset()
		//show only non zero partition entries
		if offset == 0 {
			continue
		}
		fmt.Printf("%s\n", partition.GetInfo())
	}

}

func (disk Disk) CollectedUnallocated(blocks chan<- []byte) {
	for _, partition := range disk.Partitions {

		vol := partition.GetVolume()
		if vol == nil {
			continue
		}
		bytesPerSector := int(vol.GetBytesPerSector())
		partitionOffsetB := int64(partition.GetOffset()) * int64(bytesPerSector)

		vol.CollectUnallocated(disk.Handler, partitionOffsetB, blocks)
	}
}
