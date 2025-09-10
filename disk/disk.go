package disk

import (
	"errors"
	"fmt"
	"path"
	"strings"
	"sync"

	metadata "github.com/aarsakian/FileSystemForensics/FS"
	vssLib "github.com/aarsakian/FileSystemForensics/FS/NTFS/VSS"
	UsnJrnl "github.com/aarsakian/FileSystemForensics/FS/NTFS/usnjrnl"
	gptLib "github.com/aarsakian/FileSystemForensics/disk/partition/GPT"
	mbrLib "github.com/aarsakian/FileSystemForensics/disk/partition/MBR"
	"github.com/aarsakian/FileSystemForensics/disk/volume"
	"github.com/aarsakian/FileSystemForensics/logger"
	"github.com/aarsakian/FileSystemForensics/readers"
	"github.com/aarsakian/FileSystemForensics/utils"
)

var ErrNTFSVol = errors.New("NTFS volume discovered instead of MBR")

type Partition interface {
	GetOffset() uint64
	LocateVolume(readers.DiskReader)
	GetVolume() volume.Volume
	GetInfo() string
	GetVolInfo() string
}

type Disk struct {
	MBR        *mbrLib.MBR
	GPT        *gptLib.GPT
	Handler    readers.DiskReader
	Partitions []Partition
}

func (disk *Disk) Initialize(evidencefile string, physicaldrive int, vmdkfile string) {
	var hD readers.DiskReader
	if evidencefile != "" {
		extension := path.Ext(evidencefile)
		if strings.ToLower(extension) == ".e01" {
			hD = readers.GetHandler(evidencefile, "ewf")
		} else {
			hD = readers.GetHandler(evidencefile, "raw")
		}

	} else if physicaldrive != -1 {

		hD = readers.GetHandler(fmt.Sprintf("\\\\.\\PHYSICALDRIVE%d", physicaldrive), "physicalDrive")

	} else {

		hD = readers.GetHandler(vmdkfile, "vmdk")

	}
	disk.Handler = hD
}

func (disk *Disk) Process(partitionNum int, MFTentries []int, fromMFTEntry int, toMFTEntry int) (map[int][]metadata.Record, error) {

	err := disk.DiscoverPartitions()
	if errors.Is(err, ErrNTFSVol) {
		msg := "No MBR discovered, instead NTFS volume found at 1st sector"
		fmt.Printf("%s\n", msg)
		logger.FSLogger.Warning(msg)

		disk.CreatePseudoMBR("NTFS")
	}
	disk.ProcessPartitions(partitionNum)

	disk.DiscoverFileSystems(MFTentries, fromMFTEntry, toMFTEntry)
	return disk.GetFileSystemMetadata(), err
}

func (disk Disk) GetLogicalToPhysicalMap(partitioNum int) (map[uint64]metadata.Chunk, error) {
	partition := disk.Partitions[partitioNum]
	vol := partition.GetVolume()
	if vol == nil {
		msg := fmt.Sprintf("No Volume found for partition %d", partitioNum)
		fmt.Printf("%s\n", msg)
		logger.FSLogger.Error(msg)
		return nil, errors.New(msg)
	}
	return vol.GetLogicalToPhysicalMap(), nil
}

func (disk Disk) ProcessJrnl(recordsPerPartition map[int][]metadata.Record, partitionNum int) []UsnJrnl.Record {
	var usnrecords []UsnJrnl.Record

	for partitionID, records := range recordsPerPartition {
		if partitionNum != -1 && partitionID != partitionNum {
			continue
		}
		for _, record := range metadata.FilterByName(records, "$UsnJrnl") {
			recordsCH := make(chan UsnJrnl.Record)
			wg := new(sync.WaitGroup)
			wg.Add(2)
			dataClusters := make(chan []byte, record.GetLogicalFileSize())

			go disk.AsyncWorker(wg, record, dataClusters, partitionID)
			go UsnJrnl.AsyncProcess(wg, dataClusters, recordsCH)
			for record := range recordsCH {
				usnrecords = append(usnrecords, record)
			}

			wg.Wait()
		}
	}

	return usnrecords

}

func (disk Disk) ProcessVSS(partitionID int) {
	for idx := range disk.Partitions {
		if idx != partitionID {
			continue
		}
		partitionOffsetB := int64(disk.Partitions[idx].GetOffset() * 512)

		shadowVol := new(vssLib.ShadowVolume)
		shadowVol.Process(disk.Handler, partitionOffsetB)
	}
}

func (disk Disk) Close() {
	disk.Handler.CloseHandler()
}

func (disk Disk) hasProtectiveMBR() bool {
	return disk.MBR.IsProtective()
}

func (disk *Disk) DiscoverFileSystems(MFTentries []int, fromMFTEntry int, toMFTEntry int) {
	for idx := range disk.Partitions {

		vol := disk.Partitions[idx].GetVolume()
		if vol == nil {
			continue
		}
		partitionOffsetB := int64(disk.Partitions[idx].GetOffset() * 512)
		fmt.Printf("Processing partition %d at %d ================================================\n",
			idx+1, partitionOffsetB)
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
	if utils.Hexify(mbr.Signature[:]) != "55aa" {
		return errors.New("mbr not valid")
	}
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

func (disk *Disk) DiscoverPartitions() error {

	err := disk.populateMBR()
	if err != nil {
		return err
	}
	if disk.hasProtectiveMBR() {
		disk.populateGPT()
		for idx := range disk.GPT.Partitions {

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
		if partitionNum != -1 && partitionNum != idx {
			continue
		}
		disk.Partitions[idx].LocateVolume(disk.Handler)

		parttionOffset := disk.Partitions[idx].GetOffset()
		vol := disk.Partitions[idx].GetVolume()
		if vol == nil {
			msg := "No Known Volume at partition %d (Currently supported NTFS BTRFS)."
			logger.FSLogger.Error(fmt.Sprintf(msg, idx))
			continue //fs not found
		}
		msg := "Partition %d  %s at %d sector"
		fmt.Printf(msg+"\n", idx+1, vol.GetSignature(), parttionOffset)
		logger.FSLogger.Info(fmt.Sprintf(msg, idx+1, vol.GetSignature(), parttionOffset))

	}

}

func (disk Disk) GetFileSystemMetadata() map[int][]metadata.Record {

	recordsPerPartition := map[int][]metadata.Record{}
	for idx, partition := range disk.Partitions {

		vol := partition.GetVolume()
		if vol == nil {
			continue
		}
		recordsPerPartition[idx] = vol.GetFS()

	}
	return recordsPerPartition
}

func (disk Disk) AsyncWorker(wg *sync.WaitGroup, record metadata.Record, dataClusters chan<- []byte, partitionNum int) {
	defer wg.Done()
	partition := disk.Partitions[partitionNum]

	vol := partition.GetVolume()
	sectorsPerCluster := int(vol.GetSectorsPerCluster())
	bytesPerSector := int(vol.GetBytesPerSector())
	partitionOffsetB := int64(partition.GetOffset()) * int64(bytesPerSector)

	if record.IsFolder() {
		msg := fmt.Sprintf("Record %s Id %d is folder! No data to export.", record.GetFname(), record.GetID())
		logger.FSLogger.Warning(msg)
		close(dataClusters)
		return
	}
	fmt.Printf("pulling data file %s Id %d\n", record.GetFname(), record.GetID())
	linkedRecords := record.GetLinkedRecords()
	if len(linkedRecords) == 0 {
		record.LocateDataAsync(disk.Handler, partitionOffsetB, sectorsPerCluster*bytesPerSector, dataClusters)
	} else { // attribute runlist

		for _, linkedRecord := range linkedRecords {
			linkedRecord.LocateDataAsync(disk.Handler, partitionOffsetB, sectorsPerCluster*bytesPerSector, dataClusters)

		}
	}
	// use lsize to make sure that we cannot exceed the logical size

	close(dataClusters)

}
func (disk Disk) Worker(wg *sync.WaitGroup, records []metadata.Record, results chan<- utils.AskedFile, partitionNum int) {
	defer wg.Done()
	partition := disk.Partitions[partitionNum]

	vol := partition.GetVolume()
	sectorsPerCluster := int(vol.GetSectorsPerCluster())
	bytesPerSector := int(vol.GetBytesPerSector())
	partitionOffsetB := int64(partition.GetOffset())*512 + vol.GetFSOffset()

	physicalToLogicalMap, err := disk.GetLogicalToPhysicalMap(partitionNum)
	if err != nil {
		return
	}

	for _, record := range records {

		if record.IsFolder() {
			msg := fmt.Sprintf("Record %s Id %d is folder! No data to export.", record.GetFname(), record.GetID())
			logger.FSLogger.Warning(msg)
			continue
		}

		fmt.Printf("pulling data file %s Id %d\n", record.GetFname(), record.GetID())
		linkedRecords := record.GetLinkedRecords()
		if len(linkedRecords) == 0 {
			record.LocateData(disk.Handler, partitionOffsetB, sectorsPerCluster*bytesPerSector, results, physicalToLogicalMap)
		} else { // attribute runlist

			for _, linkedRecord := range linkedRecords {
				linkedRecord.LocateData(disk.Handler, partitionOffsetB, sectorsPerCluster*bytesPerSector, results, physicalToLogicalMap)

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

func (disk Disk) ListUnallocated() {
	for _, partition := range disk.Partitions {
		vol := partition.GetVolume()
		if vol == nil {
			continue
		}
		unallocatedClusters := vol.GetUnallocatedClusters()

		for _, unallocatedCluster := range unallocatedClusters {
			fmt.Printf("%d \t", unallocatedCluster)
		}

		fmt.Printf("Total unallocated clusters %d", len(unallocatedClusters))
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

		unallocatedClusters := vol.GetUnallocatedClusters()
		blockSize := 1 // nof consecutive clusters
		prevClusterOffset := unallocatedClusters[0]

		ntfs := vol.(*volume.NTFS)

		for idx, unallocatedCluster := range unallocatedClusters {
			if idx == 0 {
				continue
			} else if idx == len(unallocatedClusters)-1 {
				blockSize += 1
			}

			if unallocatedCluster-prevClusterOffset <= 1 && idx != len(unallocatedClusters)-1 { //last one break
				blockSize += 1
			} else {

				firstBlockCluster := unallocatedClusters[idx-blockSize]
				offset := partitionOffsetB + int64(firstBlockCluster)*int64(ntfs.VBR.SectorsPerCluster)*int64(ntfs.VBR.BytesPerSector)

				blocks <- disk.Handler.ReadFile(offset, blockSize*int(ntfs.VBR.SectorsPerCluster)*int(ntfs.VBR.BytesPerSector))
				blockSize = 1

			}
			prevClusterOffset = unallocatedCluster

		}
		close(blocks)

	}

}
