package volume

import (
	"bytes"
	"fmt"
	"math"
	"time"

	metadata "github.com/aarsakian/FileSystemForensics/FS"
	"github.com/aarsakian/FileSystemForensics/FS/NTFS/MFT"
	"github.com/aarsakian/FileSystemForensics/img"
	"github.com/aarsakian/FileSystemForensics/logger"
	"github.com/aarsakian/FileSystemForensics/utils"
)

type NTFS struct {
	VBR *VBR
	MFT *MFT.MFTTable
}

type VBR struct { //Volume Boot Record
	JumpInstruction   [3]byte //0-3
	Signature         [4]byte //4 bytes NTFS 3-7
	NotUsed1          [4]byte
	BytesPerSector    uint16   // 11-13
	SectorsPerCluster uint8    //13
	NotUsed2          [26]byte //13-39
	TotalSectors      uint64   //39-47
	MFTOffset         uint64   //48-56
	MFTMirrOffset     uint64   //56-64
}

func (ntfs *NTFS) AddVolume(data []byte) {
	ntfs.VBR = new(VBR)
	ntfs.VBR.Parse(data)
}

func (ntfs *NTFS) Process(hD img.DiskReader, partitionOffsetB int64, MFTSelectedEntries []int,
	fromMFTEntry int, toMFTEntry int) {
	physicalOffset := partitionOffsetB + int64(ntfs.VBR.MFTOffset)*int64(ntfs.VBR.SectorsPerCluster)*int64(ntfs.VBR.BytesPerSector)

	length := int(1024) // len of MFT record

	msg := "Reading first record entry to determine the size of $MFT Table at offset %d"
	fmt.Printf(msg+"\n", physicalOffset)
	logger.FSLogger.Info(fmt.Sprintf(msg, physicalOffset))

	data := hD.ReadFile(physicalOffset, length)

	ntfs.MFT = new(MFT.MFTTable)
	ntfs.MFT.ProcessRecords(data)
	ntfs.MFT.DetermineClusterOffsetLength()

	// fill buffer before parsing the record

	MFTAreaBuf := ntfs.CollectMFTArea(hD, partitionOffsetB)
	start := time.Now()
	ntfs.ProcessMFT(MFTAreaBuf, MFTSelectedEntries, fromMFTEntry, toMFTEntry)
	fmt.Println("completed at ", time.Since(start))

	start = time.Now()
	msg = "Linking $MFT record non resident $MFT entries"
	fmt.Printf("%s\n", msg)
	logger.FSLogger.Info(msg)

	ntfs.MFT.CreateLinkedRecords()
	fmt.Printf("completed at %f secs \n", time.Since(start).Seconds())

	start = time.Now()
	fmt.Printf("Processing NoN resident attributes of %d records.\n", len(ntfs.MFT.Records))
	ntfs.MFT.ProcessNonResidentRecords(hD, partitionOffsetB, int(ntfs.VBR.SectorsPerCluster)*int(ntfs.VBR.BytesPerSector))
	elapsed := time.Since(start)

	fmt.Printf("completed at %fsecs \n", elapsed.Seconds())

	if len(MFTSelectedEntries) == 0 && fromMFTEntry == -1 && toMFTEntry == math.MaxUint32 { // additional processing only when user has not selected entries

		start = time.Now()

		msg = "Locating parent $MFT records from Filename attributes"
		fmt.Printf("%s\n", msg)
		logger.FSLogger.Info(msg)
		ntfs.MFT.FindParentRecords()

		fmt.Printf("completed at  %f secs \n", time.Since(start).Seconds())

		start = time.Now()

		msg = "Calculating files sizes from $I30"
		fmt.Printf("%s\n", msg)
		logger.FSLogger.Info(msg)
		ntfs.MFT.CalculateFileSizes()

		fmt.Println("completed at ", time.Since(start).Seconds())

	}

}

func (vbr *VBR) Parse(data []byte) {
	utils.Unmarshal(data, vbr)
}

func (ntfs NTFS) GetFS() []metadata.Record {
	//explicit conversion
	var records []metadata.Record
	for _, record := range ntfs.MFT.Records {
		temp := record
		records = append(records, metadata.NTFSRecord{&temp})
	}
	return records
}

func (ntfs NTFS) GetInfo() string {
	return fmt.Sprintf("%s size %d cluster size %d", ntfs.GetSignature(), ntfs.VBR.TotalSectors*uint64(ntfs.VBR.BytesPerSector),
		ntfs.VBR.SectorsPerCluster)
}

func (ntfs NTFS) GetUnallocatedClusters() []int {
	bitmapRecord := ntfs.MFT.Records[6]
	return bitmapRecord.GetUnallocatedClusters()
}

func (ntfs *NTFS) ProcessMFT(data []byte, MFTSelectedEntries []int,
	fromMFTEntry int, toMFTEntry int) {

	totalRecords := len(data) / MFT.RecordSize
	var buf bytes.Buffer
	if fromMFTEntry != -1 {
		totalRecords -= fromMFTEntry
	}
	if fromMFTEntry > totalRecords {
		panic("MFT start entry exceeds $MFT number of records")
	}

	if toMFTEntry != math.MaxUint32 && toMFTEntry > totalRecords {
		panic("MFT end entry exceeds $MFT number of records")
	}
	if toMFTEntry != math.MaxUint32 {
		totalRecords -= toMFTEntry
	}
	if len(MFTSelectedEntries) > 0 {
		totalRecords = len(MFTSelectedEntries)
	}
	buf.Grow(totalRecords * MFT.RecordSize)

	for i := 0; i < len(data); i += MFT.RecordSize {
		if i/MFT.RecordSize > toMFTEntry {
			break
		}
		for _, MFTSelectedEntry := range MFTSelectedEntries {

			if i/MFT.RecordSize != MFTSelectedEntry {

				continue
			}

			buf.Write(data[i : i+MFT.RecordSize])

		}
		//buffer full break

		if fromMFTEntry > i/MFT.RecordSize {
			continue
		}
		if len(MFTSelectedEntries) == 0 {
			buf.Write(data[i : i+MFT.RecordSize])
		}
		if buf.Len() == len(MFTSelectedEntries)*MFT.RecordSize {
			break
		}

	}
	ntfs.MFT.ProcessRecordsAsync(buf.Bytes())

}

func (ntfs NTFS) CollectMFTArea(hD img.DiskReader, partitionOffsetB int64) []byte {
	var buf bytes.Buffer

	length := int(ntfs.MFT.Size) * int(ntfs.VBR.BytesPerSector) * int(ntfs.VBR.SectorsPerCluster) // allow for MFT size
	buf.Grow(length)

	runlist := ntfs.MFT.Records[0].GetRunList("DATA") // first record $MFT
	offset := 0

	for runlist != nil {
		offset += int(runlist.Offset)

		clusters := int(runlist.Length)

		data := hD.ReadFile(partitionOffsetB+int64(offset)*int64(ntfs.VBR.SectorsPerCluster)*int64(ntfs.VBR.BytesPerSector), clusters*int(ntfs.VBR.BytesPerSector)*int(ntfs.VBR.SectorsPerCluster))
		buf.Write(data)

		if runlist.Next == nil {
			break
		}

		runlist = runlist.Next
	}
	return buf.Bytes()
}

func (vbr VBR) GetSignature() string {
	return string(vbr.Signature[:])
}

func (ntfs NTFS) GetSignature() string {
	return ntfs.VBR.GetSignature()
}

func (ntfs NTFS) HasValidSignature() bool {
	return ntfs.VBR.GetSignature() == "NTFS"
}
func (ntfs NTFS) GetSectorsPerCluster() int {
	return int(ntfs.VBR.SectorsPerCluster)
}
func (ntfs NTFS) GetBytesPerSector() uint64 {
	return uint64(ntfs.VBR.BytesPerSector)
}
