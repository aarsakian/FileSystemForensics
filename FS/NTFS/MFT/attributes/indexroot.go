package attributes

import (
	"fmt"
	"sort"

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
	ChildVCN   int64
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

func (idxRoot IndexRoot) GetIndexEntriesSortedByMFTEntry() IndexEntries {
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

func (idxEntries ByMFTEntryID) Len() int { return len(idxEntries) }
func (idxEntries ByMFTEntryID) Less(i, j int) bool {
	return idxEntries[i].Fnattr.ParRef < idxEntries[j].Fnattr.ParRef
}
func (idxEntries ByMFTEntryID) Swap(i, j int) {
	idxEntries[i], idxEntries[j] = idxEntries[j], idxEntries[i]
}
