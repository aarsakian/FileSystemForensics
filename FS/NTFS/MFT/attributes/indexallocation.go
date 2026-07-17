package attributes

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/aarsakian/FileSystemForensics/logger"
	"github.com/aarsakian/FileSystemForensics/utils"
)

type FixUp struct {
	Signature      []byte
	OriginalValues [][]byte
}

type NodeHeader struct {
	OffsetEntryList          uint32 // 0-4 offset to start of the index entry
	OffsetEndUsedEntryList   uint32 //4-8 where EntryList ends relative to the start of node header
	OffsetEndEntryListBuffer uint32 //8-12
	Flags                    uint32 //12-16 0x01 no children
}

type IndexAllocationRecords struct {
	Header  *AttributeHeader
	Records []IndexAllocation
}

type IndexAllocation struct {
	Signature        [4]byte //0-4
	FixupArrayOffset uint16  //4-6
	NumFixupEntries  uint16  //6-8
	LSN              int64   //8-16
	VCN              int64   //16-24 where the record fits in the tree
	Nodeheader       *NodeHeader
	IndexEntries     IndexEntries
	FixUp            *FixUp
}

func (idxAlloaction IndexAllocation) GetEntries() IndexEntries {
	return idxAlloaction.IndexEntries
}

func (idxAllocation IndexAllocation) GetSignature() string {
	return string(idxAllocation.Signature[:])
}

func (idxAllocation *IndexAllocationRecords) SetHeader(header *AttributeHeader) {
	idxAllocation.Header = header
}

func (idxAllocationRecs IndexAllocationRecords) GetHeader() AttributeHeader {
	return *idxAllocationRecs.Header
}

func (idxAllocationRecs IndexAllocationRecords) FindType() string {
	return idxAllocationRecs.Header.GetType()
}

func (idxAllocationRecs IndexAllocationRecords) IsNoNResident() bool {
	return idxAllocationRecs.Header.IsNoNResident()
}

func (idxAllocationRecs IndexAllocationRecords) GetEntries() IndexEntries {
	var idxEntries IndexEntries
	for _, record := range idxAllocationRecs.Records {
		idxEntries = append(idxEntries, record.GetEntries()...)
	}
	return idxEntries
}

func (idxAllocationRecs IndexAllocationRecords) GetInfo() string {
	var txt strings.Builder

	for _, record := range idxAllocationRecs.Records {
		for _, idxEntry := range record.IndexEntries {
			txt.WriteString(fmt.Sprintf(" %s", idxEntry.GetInfo()))
		}
	}
	return txt.String()

}

func (idxAllocation *IndexAllocationRecords) Parse(data []byte) {
	const recordSize = 4096
	if len(data) < recordSize {
		idxAllocation.Records = nil
		return
	}

	idxAllocationRecords := make([]IndexAllocation, 0, len(data)/recordSize)
	for offset := 0; offset+recordSize <= len(data); offset += recordSize {
		chunk := data[offset : offset+recordSize]
		if chunk[0] != 'I' || chunk[1] != 'N' || chunk[2] != 'D' || chunk[3] != 'X' {
			break
		}

		idxAllocationRecord := IndexAllocation{}
		err := idxAllocationRecord.Parse(chunk)
		if err != nil {
			break
		}
		idxAllocationRecords = append(idxAllocationRecords, idxAllocationRecord)
	}
	idxAllocation.Records = idxAllocationRecords
}

func (idxAllocation *IndexAllocation) Parse(data []byte) error {
	utils.Unmarshal(data[:24], idxAllocation)

	if idxAllocation.GetSignature() == "INDX" {
		err := idxAllocation.ProcessFixUpArrays(data)
		if err != nil {
			return err
		}
		var nodeheader *NodeHeader = new(NodeHeader)
		utils.Unmarshal(data[24:24+16], nodeheader)
		idxAllocation.Nodeheader = nodeheader

		idxEntryOffset := nodeheader.OffsetEntryList + 24 // relative to the start of node header

		//needs check normally should be compared with data
		for sectorNum := 1; sectorNum*512 <= len(data); sectorNum++ {
			if data[sectorNum*512-2] == idxAllocation.FixUp.Signature[0] && data[sectorNum*512-1] == idxAllocation.FixUp.Signature[1] {
				data[sectorNum*512-2] = idxAllocation.FixUp.OriginalValues[sectorNum-1][0]
				data[sectorNum*512-1] = idxAllocation.FixUp.OriginalValues[sectorNum-1][1]

			}
		}

		if nodeheader.OffsetEndUsedEntryList > idxEntryOffset { // only when available exceeds start offset parse
			if nodeheader.OffsetEndUsedEntryList > uint32(len(data)) {
				msg := fmt.Sprintf("data buffer exceed by %d in parsing index allocation entry",
					nodeheader.OffsetEndUsedEntryList-uint32(len(data)))
				logger.FSLogger.Warning(msg)
				return errors.New(msg)
			}
			idxAllocation.IndexEntries = Parse(data[idxEntryOffset : nodeheader.OffsetEndUsedEntryList+24])
		}

	} else {
		msg := "INDX signature not found in index allocation attribute"
		logger.FSLogger.Warning(msg)
		return errors.New(msg)
	}
	return nil
}

func (idxAllocation *IndexAllocation) ProcessFixUpArrays(data []byte) error {
	if len(data) < int(idxAllocation.FixupArrayOffset) ||
		len(data) < int(idxAllocation.FixupArrayOffset+2*idxAllocation.NumFixupEntries) {

		msg := fmt.Sprintf("Data not enough to parse fixup array by %d", int(2*idxAllocation.NumFixupEntries)-len(data))
		logger.FSLogger.Warning(msg)
		return errors.New(msg)
	} else if idxAllocation.FixupArrayOffset > idxAllocation.FixupArrayOffset+2*idxAllocation.NumFixupEntries {
		msg := fmt.Sprintf("fixup array offset issue %d vs %d", idxAllocation.FixupArrayOffset, int(2*idxAllocation.NumFixupEntries)-len(data))
		logger.FSLogger.Warning(msg)
		return errors.New(msg)
	}

	fixuparray := data[idxAllocation.FixupArrayOffset : idxAllocation.FixupArrayOffset+2*idxAllocation.NumFixupEntries]

	//an 2-d array consisting of numfixupentries each 2 bytes first entry is the fixup
	fixupvals := make([][]byte, idxAllocation.NumFixupEntries-1)
	pos := 0
	for val := 2; val < len(fixuparray); val = val + 2 {

		fixupvals[pos] = fixuparray[val : val+2]
		pos++
	}

	//2 byte USN 8 byte USA
	idxAllocation.FixUp = &FixUp{Signature: fixuparray[:2], OriginalValues: fixupvals}
	return nil
}

func (idxAllocation IndexAllocation) GetIndexEntriesSortedByMFTEntry() IndexEntries {
	idxEntriesSortedByMFTEntryID := utils.FilterClone(idxAllocation.IndexEntries, func(idxEntry IndexEntry) bool {
		return idxEntry.Fnattr != nil
	})

	slices.SortFunc(idxEntriesSortedByMFTEntryID, func(idxEntryA, idxEntryB IndexEntry) int {
		return int(idxEntryA.Fnattr.ParRef - idxEntryB.Fnattr.ParRef)
	})
	return idxEntriesSortedByMFTEntryID
}

func (idxAlloactionRecs IndexAllocationRecords) GetIndexEntriesSortedByMFTEntry() IndexEntries {
	var idxEntries IndexEntries
	for _, record := range idxAlloactionRecs.Records {
		idxEntries = append(idxEntries, record.GetIndexEntriesSortedByMFTEntry()...)
	}
	return idxEntries
}

func (nodeheader *NodeHeader) Parse(data []byte) error {
	utils.Unmarshal(data, nodeheader)

	if nodeheader.OffsetEndUsedEntryList == 0 {
		msg := "index root has zero used buffer as derived from nodeheader"
		logger.FSLogger.Warning(msg)
		return errors.New(msg)

	}
	return nil
}
