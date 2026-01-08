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
	linkedRecords := make([]Record, 0, len(record.FileDirEntry.GetLinkedRecords()))
	for _, linkedRecord := range record.FileDirEntry.GetLinkedRecords() {
		linkedRecords = append(linkedRecords, BTRFSRecord{linkedRecord})
	}
	return linkedRecords
}

func (ntfsRecord NTFSRecord) GetLinkedRecords() []Record {
	linkedRecords := make([]Record, 0, len(ntfsRecord.Record.LinkedRecords))
	for _, linkedRecord := range ntfsRecord.Record.LinkedRecords {
		linkedRecords = append(linkedRecords, NTFSRecord{linkedRecord})
	}
	return linkedRecords
}

func (ntfsRecord NTFSRecord) FindAttribute(attrName string) Attribute {
	return ntfsRecord.Record.FindAttribute(attrName)
}

func (record BTRFSRecord) LocateDataORG(hD readers.DiskReader, partitionOffsetB int64, blockSizeB int, results chan<- utils.AskedFile,
	physicalToLogicalMap map[uint64]Chunk) {

	var buf bytes.Buffer

	lSize := int(record.GetLogicalFileSize())
	buf.Grow(lSize)

	/*if fileDirEntry.HasResidentDataAttr() {
		buf.Write(fileDirEntry.GetResidentData())

	} else {*/

	diskSize := hD.GetDiskSize()
	var volumeOffset uint64
	totalReadBytes := 0
	for _, extent := range record.GetExtents() {
		for keyOffset, chunk := range physicalToLogicalMap {
			chunkItem := chunk.(BTRFSChunk)
			if extent.ExtentDataRem.LogicalAddress >= keyOffset &&
				extent.ExtentDataRem.LogicalAddress-keyOffset < chunkItem.Size {

				blockGroupOffset := extent.ExtentDataRem.LogicalAddress - keyOffset
				volumeOffset = chunkItem.Stripes[0].Offset + blockGroupOffset
				msg := fmt.Sprintf("block group offset %d chunk offset %d", blockGroupOffset, chunkItem.Stripes[0].Offset)
				logger.FSLogger.Info(msg)
				break
			}
		}
		offset := partitionOffsetB + int64(volumeOffset) + int64(extent.ExtentDataRem.Offset)
		if offset > diskSize {
			msg := fmt.Sprintf("skipped offset %d exceeds disk size! exiting", offset)
			logger.FSLogger.Warning(msg)
			break
		}
		totalReadBytes += int(extent.ExtentDataRem.LSize)
		if extent.ExtentDataRem.LogicalAddress != 0 && extent.ExtentDataRem.LSize > 0 {
			data, _ := hD.ReadFile(offset, int(extent.ExtentDataRem.LSize))
			buf.Write(data)

			msg := fmt.Sprintf("Read %d bytes from physical offset %d (sectors) logical %d (sectors) total bytes read %d out of %d",
				extent.ExtentDataRem.LSize, offset/512, (offset-partitionOffsetB)/512, totalReadBytes, lSize)
			logger.FSLogger.Info(msg)
		} else if extent.ExtentDataRem.LogicalAddress == 0 && extent.ExtentDataRem.LSize > 0 {
			//sparse not allocated generate zeros

			buf.Write(make([]byte, extent.ExtentDataRem.LSize))
			msg := fmt.Sprintf("sparse  len %d cl.", extent.ExtentDataRem.LogicalAddress/uint64(blockSizeB))
			logger.FSLogger.Info(msg)
		}
	}
}

func (record BTRFSRecord) LocateData(hD readers.DiskReader, partitionOffsetB int64,
	blockSizeB int, dataToRead []byte,
	physicalToLogicalMap map[uint64]Chunk) {

	/*if fileDirEntry.HasResidentDataAttr() {
		buf.Write(fileDirEntry.GetResidentData())

	} else {*/

	diskSize := hD.GetDiskSize()
	var volumeOffset uint64
	totalReadBytes := 0
	for _, extents := range record.GetGroupedExtents() {
		startFrom := extents[0].ExtentDataRem.LogicalAddress
		toRead := extents[0].ExtentDataRem.LSize
		extentOffset := extents[0].ExtentDataRem.Offset

		for _, extent := range extents[1:] {
			toRead += extent.ExtentDataRem.LSize
		}

		for keyOffset, chunk := range physicalToLogicalMap {
			chunkItem := chunk.(BTRFSChunk)
			if startFrom >= keyOffset &&
				startFrom-keyOffset < chunkItem.Size {
				blockGroupOffset := startFrom - keyOffset
				volumeOffset = chunkItem.Stripes[0].Offset + blockGroupOffset
				msg := fmt.Sprintf("block group offset %d chunk offset %d", blockGroupOffset, chunkItem.Stripes[0].Offset)
				logger.FSLogger.Info(msg)
				break
			}
		}
		offset := partitionOffsetB + int64(volumeOffset) + int64(extentOffset)
		if offset > diskSize {
			msg := fmt.Sprintf("skipped offset %d exceeds disk size! exiting", offset)
			logger.FSLogger.Warning(msg)
			break
		}

		if startFrom != 0 && toRead > 0 {
			data, _ := hD.ReadFile(offset, int(toRead))
			copy(dataToRead[totalReadBytes:], data)

			msg := fmt.Sprintf("Read %d bytes from physical offset %d (sectors) logical %d (sectors) total bytes read %d ",
				toRead, offset/512, (offset-partitionOffsetB)/512, totalReadBytes)
			logger.FSLogger.Info(msg)
		} else if startFrom == 0 && toRead > 0 {
			//sparse not allocated generate zeros

			copy(dataToRead[totalReadBytes:], make([]byte, toRead))
			msg := fmt.Sprintf("sparse  len %d cl.", startFrom/uint64(blockSizeB))
			logger.FSLogger.Info(msg)
		}
		totalReadBytes += int(toRead)

	}

	//truncate buf grows over len?
	//results <- utils.AskedFile{Fname: record.GetFname(), Content: buf.Bytes()[:lSize], Id: int(record.Id)}
}

func (record NTFSRecord) LocateData(hD readers.DiskReader, partitionOffset int64,
	clusterSizeB int, dataToRead []byte,
	physicalToLogicalMap map[uint64]Chunk) {
	p := message.NewPrinter(language.Greek)

	writeOffset := 0

	if record.HasResidentDataAttr() {
		copy(dataToRead, record.GetResidentData())

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
				data, _ := hD.ReadFile(offset, int(runlist.Length)*clusterSizeB)
				copy(dataToRead[writeOffset:], data)
				res := p.Sprintf("%d", (offset-partitionOffset)/int64(clusterSizeB))

				msg := fmt.Sprintf("offset %s cl len %d cl. write offset %d", res, runlist.Length, writeOffset)
				logger.FSLogger.Info(msg)
			} else if runlist.Offset == 0 && runlist.Length > 0 {
				//sparse not allocated generate zeros
				copy(dataToRead[writeOffset:], make([]byte, int(runlist.Length)*clusterSizeB))
				msg := fmt.Sprintf("sparse  len %d cl.", runlist.Length)
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

}
