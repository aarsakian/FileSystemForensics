package metadata

import (
	"bytes"
	"fmt"

	fstree "github.com/aarsakian/FileSystemForensics/FS/BTRFS"
	attr "github.com/aarsakian/FileSystemForensics/FS/BTRFS/attributes"
	"github.com/aarsakian/FileSystemForensics/FS/NTFS/MFT"
	"github.com/aarsakian/FileSystemForensics/logger"
	"github.com/aarsakian/FileSystemForensics/readers"
	"github.com/aarsakian/FileSystemForensics/utils"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type NTFSRecord struct {
	*MFT.Record
}

type BTRFSChunk struct {
	*attr.ChunkItem
}

type BTRFSRecord struct {
	*fstree.FileDirEntry
}

func (record BTRFSRecord) FindAttribute(attrName string) Attribute {
	return record.FileDirEntry.FindAttribute(attrName)
}

func (btrfsRecord *BTRFSRecord) LocatePhysicalOffset() {

}

func (record BTRFSRecord) GetLinkedRecords() []Record {
	var linkedRecords []Record
	for _, linkedRecord := range record.FileDirEntry.GetLinkedRecords() {
		linkedRecords = append(linkedRecords, BTRFSRecord{linkedRecord})
	}
	return linkedRecords
}

func (ntfsRecord NTFSRecord) GetLinkedRecords() []Record {
	var linkedRecords []Record
	for _, linkedRecord := range ntfsRecord.Record.LinkedRecords {
		linkedRecords = append(linkedRecords, NTFSRecord{linkedRecord})
	}
	return linkedRecords
}

func (ntfsRecord NTFSRecord) FindAttribute(attrName string) Attribute {
	return ntfsRecord.Record.FindAttribute(attrName)
}

func (record BTRFSRecord) LocateData(hD readers.DiskReader, partitionOffsetB int64, blockSizeB int, results chan<- utils.AskedFile,
	physicalToLogicalMap map[uint64]Chunk) {
	p := message.NewPrinter(language.Greek)

	var buf bytes.Buffer

	lSize := int(record.GetLogicalFileSize())
	buf.Grow(lSize)

	/*if fileDirEntry.HasResidentDataAttr() {
		buf.Write(fileDirEntry.GetResidentData())

	} else {*/

	diskSize := hD.GetDiskSize()
	var volumeOffset uint64

	for _, extent := range record.GetExtents() {

		for keyOffset, chunk := range physicalToLogicalMap {
			chunkItem := chunk.(BTRFSChunk)
			if extent.ExtentDataRem.LogicaAddress >= keyOffset &&
				extent.ExtentDataRem.LogicaAddress-keyOffset < chunkItem.Size {
				blockGroupOffset := extent.ExtentDataRem.LogicaAddress - keyOffset
				volumeOffset = chunkItem.Stripes[0].Offset + blockGroupOffset
				break
			}
		}
		offset := partitionOffsetB + int64(volumeOffset) + int64(extent.ExtentDataRem.Offset)
		if offset > diskSize {
			msg := fmt.Sprintf("skipped offset %d exceeds disk size! exiting", offset)
			logger.FSLogger.Warning(msg)
			break
		}

		if extent.ExtentDataRem.LogicaAddress != 0 && extent.ExtentDataRem.LSize > 0 {
			buf.Write(hD.ReadFile(offset, int(extent.ExtentDataRem.LSize)))
			res := p.Sprintf("%d", (offset-partitionOffsetB)/int64(blockSizeB))

			msg := fmt.Sprintf("offset %s cl len %d cl.", res, extent.ExtentDataRem.LSize/uint64(blockSizeB))
			logger.FSLogger.Info(msg)
		} else if extent.ExtentDataRem.LogicaAddress == 0 && extent.ExtentDataRem.LSize > 0 {
			//sparse not allocated generate zeros

			buf.Write(make([]byte, extent.ExtentDataRem.LSize))
			msg := fmt.Sprintf("sparse  len %d cl.", extent.ExtentDataRem.LSize/uint64(blockSizeB))
			logger.FSLogger.Info(msg)
		}

	}

	//truncate buf grows over len?
	results <- utils.AskedFile{Fname: record.GetFname(), Content: buf.Bytes()[:lSize], Id: int(record.Id)}
}

func (record NTFSRecord) LocateData(hD readers.DiskReader, partitionOffset int64, clusterSizeB int, results chan<- utils.AskedFile,
	physicalToLogicalMap map[uint64]Chunk) {
	p := message.NewPrinter(language.Greek)

	writeOffset := 0

	var buf bytes.Buffer

	lSize := int(record.GetLogicalFileSize())
	buf.Grow(lSize)

	if record.HasResidentDataAttr() {
		buf.Write(record.GetResidentData())

	} else {

		runlist := record.GetRunList("DATA")

		offset := partitionOffset // partition in bytes

		diskSize := hD.GetDiskSize()

		for runlist != nil {
			offset += runlist.Offset * int64(clusterSizeB)
			if offset > diskSize {
				msg := fmt.Sprintf("skipped offset %d exceeds disk size! exiting", offset)
				logger.FSLogger.Warning(msg)
				break
			}

			if runlist.Offset != 0 && runlist.Length > 0 {
				buf.Write(hD.ReadFile(offset, int(runlist.Length)*clusterSizeB))
				res := p.Sprintf("%d", (offset-partitionOffset)/int64(clusterSizeB))

				msg := fmt.Sprintf("offset %s cl len %d cl.", res, runlist.Length)
				logger.FSLogger.Info(msg)
			}

			if runlist.Next == nil {
				break
			}

			runlist = runlist.Next
			writeOffset += int(runlist.Length) * clusterSizeB
		}

	}
	//truncate buf grows over len?
	results <- utils.AskedFile{Fname: record.GetFname(), Content: buf.Bytes()[:lSize], Id: int(record.Entry)}
}
