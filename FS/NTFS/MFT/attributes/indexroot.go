package attributes

import (
	"fmt"
	"slices"

	"github.com/aarsakian/FileSystemForensics/logger"
	"github.com/aarsakian/FileSystemForensics/utils"
)

var IndexEntryFlags = map[string]string{
	"00000001": "Child Node exists",
	"00000002": "Last Entry in list",
}

var IndexFlags = map[uint32]string{0x000001: "Has VCN", 0x000002: "Last"}

type ByMFTEntryID IndexEntries
type IndexEntries []IndexEntry

type IndexEntry struct {
	ParRef     uint64
	ParSeq     uint16
	Len        uint16 //8-9
	ContentLen uint16 //10-11
	Flags      uint32 //12-15
	ChildVCN   uint64
	Fnattr     *FNAttribute
}

type IndexRoot struct {
	Type                 [4]byte //0-3 similar to FNA type  (Hexify(Bytereverse
	CollationSortingRule [4]byte //4-7
	Sizebytes            uint32  //8-11
	Sizeclusters         uint8   //12-12
	Nodeheader           *NodeHeader
	Header               *AttributeHeader
	IndexEntries         IndexEntries
}

func (idxRoot IndexRoot) GetEntries() IndexEntries {
	return idxRoot.IndexEntries
}

func (idxEntry IndexEntry) ShowInfo() {
	if idxEntry.Fnattr != nil {
		if idxEntry.Flags == 0 {
			fmt.Printf("type %s file ref %d idx name %s  allocated size %d real size %d \n",
				idxEntry.Fnattr.GetType(), idxEntry.ParRef,
				idxEntry.Fnattr.Fname,
				idxEntry.Fnattr.AllocFsize, idxEntry.Fnattr.RealFsize)
		} else {
			fmt.Printf("type %s file ref %d idx name %s  allocated size %d real size %d VCN %d\n",
				idxEntry.Fnattr.GetType(), idxEntry.ParRef,
				idxEntry.Fnattr.Fname,
				idxEntry.Fnattr.AllocFsize, idxEntry.Fnattr.RealFsize, idxEntry.ChildVCN)
		}

	}

}

func (idxRoot *IndexRoot) SetHeader(header *AttributeHeader) {
	idxRoot.Header = header
}

func (idxRoot *IndexRoot) Parse(data []byte) {
	utils.Unmarshal(data[:12], idxRoot)

	var nodeheader *NodeHeader = new(NodeHeader)
	err := nodeheader.Parse(data[16:32])
	if err != nil {
		return
	}

	idxRoot.Nodeheader = nodeheader

	idxEntryOffset := 16 + uint16(nodeheader.OffsetEntryList)
	if int(nodeheader.OffsetEndUsedEntryList) > len(data) {
		logger.FSLogger.Warning("nodeheeader offset exceeds buffer")
		return
	}

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

func (idxRoot IndexRoot) GetIndexEntriesSortedByMFTEntry() IndexEntries {
	idxEntriesSortedByMFTEntryID := utils.FilterClone(idxRoot.IndexEntries, func(idxEntry IndexEntry) bool {
		return idxEntry.Fnattr != nil
	})

	slices.SortFunc(idxEntriesSortedByMFTEntryID, func(idxEntryA, idxEntryB IndexEntry) int {
		return int(idxEntryA.Fnattr.ParRef - idxEntryB.Fnattr.ParRef)
	})
	return idxEntriesSortedByMFTEntryID
}

func Parse(data []byte) IndexEntries {
	var idxEntries IndexEntries
	idxEntryOffset := uint16(0)
	for idxEntryOffset < uint16(len(data)) {
		var idxEntry *IndexEntry = new(IndexEntry)
		idxEntry.Parse(data[idxEntryOffset:])

		idxEntryOffset += idxEntry.Len
		if idxEntry.Len == 0 {
			logger.FSLogger.Warning("zero len index entry")
			break
		}
		idxEntries = append(idxEntries, *idxEntry)
	}
	return idxEntries

}

func (idxEntry *IndexEntry) Parse(data []byte) {
	utils.Unmarshal(data, idxEntry)

	if idxEntry.ContentLen > 0 {
		var fnattrIDXEntry FNAttribute
		utils.Unmarshal(data[16:16+uint32(idxEntry.ContentLen)],
			&fnattrIDXEntry)
		if 16+66+2*uint32(fnattrIDXEntry.Nlen) > uint32(len(data)) {
			msg := fmt.Sprintf("data buffer exceed by %d in parsing index root entry",
				16+66+2*uint32(fnattrIDXEntry.Nlen)-uint32(len(data)))
			logger.FSLogger.Warning(msg)
			return
		}
		fnattrIDXEntry.Fname = utils.DecodeUTF16(data[16+66 : 16+66+2*uint32(fnattrIDXEntry.Nlen)])
		idxEntry.Fnattr = &fnattrIDXEntry

	}
}
