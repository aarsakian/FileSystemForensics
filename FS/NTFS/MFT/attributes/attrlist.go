package attributes

import (
	"fmt"

	"github.com/aarsakian/MFTExtractor/utils"
)

type AttributeListEntries struct {
	Entries []AttributeList
	Header  *AttributeHeader
}

type AttributeList struct { //more than one MFT entry to store a file/directory its attributes
	Type       string //        typeif 0-4    # 4
	Len        uint16 //4-6
	Namelen    uint8  //7unsigned char           # 1
	Nameoffset uint8  //8-8               # 1
	StartVcn   uint64 //8-16         # 8
	ParRef     uint64 //16-22      # 6
	ParSeq     uint16 //       22-24    # 2
	ID         uint8  //     24-26   # 4
	Name       utils.NoNull
}

func (attrList AttributeList) GetType() string {
	return AttrTypes[attrList.Type]
}

func (attrListEntries *AttributeListEntries) SetHeader(header *AttributeHeader) {
	attrListEntries.Header = header
}

func (attrListEntries AttributeListEntries) GetHeader() AttributeHeader {
	return *attrListEntries.Header
}

func (attrListEntries *AttributeListEntries) Parse(data []byte) {
	attrLen := uint16(0)
	for 24+attrLen < uint16(len(data)) {
		var attrList AttributeList
		utils.Unmarshal(data[attrLen:attrLen+24], &attrList)
		attrList.Name = utils.NoNull(data[attrLen+uint16(attrList.Nameoffset) : attrLen+uint16(attrList.Nameoffset)+2*uint16(attrList.Namelen)])

		attrListEntries.Entries = append(attrListEntries.Entries, attrList)
		attrLen += attrList.Len

	}
}

func (attrListEntries AttributeListEntries) FindType() string {
	return attrListEntries.Header.GetType()
}

func (attrListEntries AttributeListEntries) IsNoNResident() bool {
	return attrListEntries.Header.IsNoNResident()
}

func (attrListEntries AttributeListEntries) ShowInfo() {
	for _, attrList := range attrListEntries.Entries {
		fmt.Printf("Attr List Type %s MFT Ref %d startVCN %d name %s \n",
			attrList.GetType(), attrList.ParRef, attrList.StartVcn, attrList.Name)
	}

}
