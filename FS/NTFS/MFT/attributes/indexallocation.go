package attributes

import (
	"fmt"
	"sort"

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

func (idxAllocationRecs IndexAllocationRecords) ShowInfo() {
	fmt.Printf("type %s \n", idxAllocationRecs.FindType())
	for _, record := range idxAllocationRecs.Records {
		fmt.Printf("nof entries  %d ", record.NumFixupEntries)
		for _, idxEntry := range record.IndexEntries {
			idxEntry.ShowInfo()
		}
	}

}

func (idxAllocation *IndexAllocationRecords) Parse(data []byte) {
	//index record size 4096bytes
	idxAllocation.Records = make([]IndexAllocation, len(data)/4096)
	for i := 0; i < len(data); i = i + 4096 {
		idxAllocation.Records[i/4096].Parse(data[i : i+4096])
	}
}

func (idxAllocation *IndexAllocation) Parse(data []byte) {
	utils.Unmarshal(data[:24], idxAllocation)

	idxAllocation.ProcessFixUpArrays(data)

	if idxAllocation.GetSignature() == "INDX" {
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
				return
			}
			idxAllocation.IndexEntries = Parse(data[idxEntryOffset : nodeheader.OffsetEndUsedEntryList+24])
		}

	} else {
		logger.FSLogger.Warning("INDX signature not found in index allocation attribute")
	}

}

func (idxAllocation *IndexAllocation) ProcessFixUpArrays(data []byte) {
	if len(data) < int(idxAllocation.FixupArrayOffset) ||
		len(data) < int(idxAllocation.FixupArrayOffset+2*idxAllocation.NumFixupEntries) {

		msg := fmt.Sprintf("Data not enough to parse fixup array by %d", int(2*idxAllocation.NumFixupEntries)-len(data))
		logger.FSLogger.Warning(msg)
		return
	} else if idxAllocation.FixupArrayOffset > idxAllocation.FixupArrayOffset+2*idxAllocation.NumFixupEntries {
		msg := fmt.Sprintf("fixup array offset issue %d vs %d", idxAllocation.FixupArrayOffset, int(2*idxAllocation.NumFixupEntries)-len(data))
		logger.FSLogger.Warning(msg)
		return
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

}

func (idxAllocation IndexAllocation) GetIndexEntriesSortedByMFTEntry() IndexEntries {
	var idxEntries IndexEntries
	for _, entry := range idxAllocation.IndexEntries {
		if entry.Fnattr == nil {
			continue
		}
		idxEntries = append(idxEntries, entry)
	}
	sort.Sort(ByMFTEntryID(idxEntries))
	return idxEntries
}

func (idxAlloactionRecs IndexAllocationRecords) GetIndexEntriesSortedByMFTEntry() IndexEntries {
	var idxEntries IndexEntries
	for _, record := range idxAlloactionRecs.Records {
		idxEntries = append(idxEntries, record.GetIndexEntriesSortedByMFTEntry()...)
	}
	return idxEntries
}
