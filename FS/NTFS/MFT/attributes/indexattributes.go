package attributes

import (
	"fmt"
	"sort"

	"github.com/aarsakian/MFTExtractor/utils"
)

var IndexFlags = map[uint32]string{0x000001: "Has VCN", 0x000002: "Last"}

type ByMFTEntryID IndexEntries
type IndexEntries []IndexEntry

type IndexEntry struct {
	ParRef     uint64
	ParSeq     uint16
	Len        uint16 //8-9
	ContentLen uint16 //10-11
	Flags      uint32 //12-15
	ChildVCN   int64
	Fnattr     *FNAttribute
}

type IndexRoot struct {
	Type                 string //0-3 similar to FNA type
	CollationSortingRule string //4-7
	Sizebytes            uint32 //8-11
	Sizeclusters         uint8  //12-12
	Nodeheader           *NodeHeader
	Header               *AttributeHeader
	IndexEntries         IndexEntries
}

type NodeHeader struct {
	OffsetEntryList          uint32 // 16-20 offset to start of the index entry
	OffsetEndUsedEntryList   uint32 //20-24 where EntryList ends
	OffsetEndEntryListBuffer uint32 //24-28
	Flags                    uint32 //0x01 no children
}

type IndexAllocation struct {
	Signature        string //0-4
	FixupArrayOffset int16  //4-6
	NumEntries       int16  //6-8
	LSN              int64  //8-16
	VCN              int64  //16-24 where the record fits in the tree
	Nodeheader       *NodeHeader
	Header           *AttributeHeader
	IndexEntries     IndexEntries
}

func (idxEntry IndexEntry) ShowInfo() {
	if idxEntry.Fnattr != nil {
		fmt.Printf("type %s file ref %d idx name %s flags %d allocated size %d real size %d \n", idxEntry.Fnattr.GetType(), idxEntry.ParRef,
			idxEntry.Fnattr.Fname, idxEntry.Flags, idxEntry.Fnattr.AllocFsize, idxEntry.Fnattr.RealFsize)
	}

}

func (idxRoot *IndexRoot) SetHeader(header *AttributeHeader) {
	idxRoot.Header = header
}

func (idxRoot *IndexRoot) Parse(data []byte) {
	utils.Unmarshal(data[:12], idxRoot)

	var nodeheader *NodeHeader = new(NodeHeader)
	utils.Unmarshal(data[16:32], nodeheader)
	idxRoot.Nodeheader = nodeheader

	idxEntryOffset := 16 + uint16(nodeheader.OffsetEntryList)

	idxRoot.IndexEntries = Parse(data[idxEntryOffset:nodeheader.OffsetEndUsedEntryList])

}

func (idxRoot IndexRoot) GetHeader() AttributeHeader {
	return *idxRoot.Header
}

func (idxRoot IndexRoot) IsNoNResident() bool {
	return false // always resident
}

func (idxRoot IndexRoot) FindType() string {
	return idxRoot.Header.GetType()
}

func (idxRoot IndexRoot) ShowInfo() {
	fmt.Printf("type %s nof entries %d\n", idxRoot.FindType(), len(idxRoot.IndexEntries))
	for _, idxEntry := range idxRoot.IndexEntries {
		idxEntry.ShowInfo()
	}
}

func (idxRoot IndexRoot) GetIndexEntriesSortedByMFTEntryID() IndexEntries {
	var idxEntries IndexEntries
	for _, entry := range idxRoot.IndexEntries {
		if entry.Fnattr == nil {
			continue
		}
		idxEntries = append(idxEntries, entry)
	}
	sort.Sort(ByMFTEntryID(idxEntries))
	return idxEntries
}

func (idxAllocation *IndexAllocation) SetHeader(header *AttributeHeader) {
	idxAllocation.Header = header
}

func (idxAllocation IndexAllocation) GetHeader() AttributeHeader {
	return *idxAllocation.Header
}

func (idxAllocation IndexAllocation) FindType() string {
	return idxAllocation.Header.GetType()
}

func (idxAllocation IndexAllocation) IsNoNResident() bool {
	return idxAllocation.Header.IsNoNResident()
}

func (idxAllocation IndexAllocation) ShowInfo() {
	fmt.Printf("type %s nof entries %d\n", idxAllocation.FindType(), idxAllocation.NumEntries)
	for _, idxEntry := range idxAllocation.IndexEntries {
		idxEntry.ShowInfo()
	}
}

func Parse(data []byte) IndexEntries {
	var idxEntries IndexEntries
	idxEntryOffset := uint16(0)
	for idxEntryOffset < uint16(len(data)) {
		var idxEntry *IndexEntry = new(IndexEntry)
		idxEntry.Parse(data[idxEntryOffset:])

		idxEntryOffset += idxEntry.Len
		idxEntries = append(idxEntries, *idxEntry)
	}
	return idxEntries

}

func (idxEntry *IndexEntry) Parse(data []byte) {
	utils.Unmarshal(data[:16], idxEntry)

	if IndexFlags[idxEntry.Flags] == "Has VCN" {
		idxEntry.ChildVCN = utils.ReadEndianInt(data[16+idxEntry.Len-8 : 16+idxEntry.Len])
	}

	if idxEntry.ContentLen > 0 {
		var fnattrIDXEntry FNAttribute
		utils.Unmarshal(data[16:16+uint32(idxEntry.ContentLen)],
			&fnattrIDXEntry)

		fnattrIDXEntry.Fname = utils.DecodeUTF16(data[16+66 : 16+66+2*uint32(fnattrIDXEntry.Nlen)])
		idxEntry.Fnattr = &fnattrIDXEntry

	}
}

func (idxAllocation *IndexAllocation) Parse(data []byte) {
	utils.Unmarshal(data[:24], idxAllocation)
	if idxAllocation.Signature == "INDX" {
		var nodeheader *NodeHeader = new(NodeHeader)
		utils.Unmarshal(data[24:24+16], nodeheader)
		idxAllocation.Nodeheader = nodeheader

		idxEntryOffset := nodeheader.OffsetEntryList + 24       // relative to the start of node header
		if nodeheader.OffsetEndUsedEntryList > idxEntryOffset { // only when available exceeds start offset parse
			idxAllocation.IndexEntries = Parse(data[idxEntryOffset:nodeheader.OffsetEndUsedEntryList])
		}

	}

}

func (idxAllocation IndexAllocation) GetIndexEntriesSortedByMFTEntryID() IndexEntries {
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

func (idxEntries ByMFTEntryID) Len() int { return len(idxEntries) }
func (idxEntries ByMFTEntryID) Less(i, j int) bool {
	return idxEntries[i].Fnattr.ParRef < idxEntries[j].Fnattr.ParRef
}
func (idxEntries ByMFTEntryID) Swap(i, j int) {
	idxEntries[i], idxEntries[j] = idxEntries[j], idxEntries[i]
}
