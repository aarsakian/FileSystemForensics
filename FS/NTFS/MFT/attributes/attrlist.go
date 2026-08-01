package attributes

import (
	"fmt"
	"math"
	"strings"

	"github.com/aarsakian/FileSystemForensics/logger"
	"github.com/aarsakian/FileSystemForensics/utils"
)

type LinkedRecordInfo struct {
	RefEntry uint32
	RefSeq   uint16
	StartVCN uint64
}

type AttributeListEntries struct {
	Entries []AttributeList
	Header  *AttributeHeader
}

type AttributeList struct { //more than one MFT entry to store a file/directory its attributes
	Type       [4]byte //        typeif 0-4    # 4
	Len        uint16  //4-6
	Namelen    uint8   //7unsigned char           # 1
	Nameoffset uint8   //8-8               # 1
	StartVcn   uint64  //8-16         # 8
	ParRef     uint64  //16-22      # 6
	ParSeq     uint16  //       22-24    # 2
	ID         uint8   //     24-26   # 4
	Name       string
}

func (attrList AttributeList) GetType() string {
	return AttrTypes[utils.ToUint32(attrList.Type[:])]
}

func (attrListEntries *AttributeListEntries) SetHeader(header *AttributeHeader) {
	attrListEntries.Header = header
}

func (attrListEntries AttributeListEntries) GetHeader() AttributeHeader {
	return *attrListEntries.Header
}

func (attrListEntries *AttributeListEntries) Parse(data []byte) {
	attrLen := uint16(0)
	for 24+attrLen < uint16(len(data)) && attrLen < math.MaxUint16-24 {

		var attrList AttributeList
		utils.Unmarshal(data[attrLen:attrLen+24], &attrList)

		start := attrLen + uint16(attrList.Nameoffset)
		end := start + 2*uint16(attrList.Namelen)

		if end > uint16(len(data)) {
			logger.FSLogger.Warning(fmt.Sprintf("AttributeListEntries: Parse: end index %d exceeds data length %d", end, len(data)))
			break
		}
		//reduce memory footprint by copying only the relevant portion of the data slice
		raw := make([]byte, end-start)
		copy(raw, data[start:end])
		attrList.Name = utils.RemoveNulls(raw)

		attrListEntries.Entries = append(attrListEntries.Entries, attrList)
		attrLen += attrList.Len
		if attrLen == 0 || attrList.Len == 0 {
			break
		}
		if attrList.Len < 24 || attrList.Len > 4096 {
			break
		}

	}
}

func (attrListEntries AttributeListEntries) GetLinkedRecordsInfo(entryID uint32) []LinkedRecordInfo {
	var linkedRecordsInfo []LinkedRecordInfo
	for _, entry := range attrListEntries.Entries {
		// attribute is stored in the same base record
		if entry.ParRef == uint64(entryID) {
			continue
		}
		logger.FSLogger.Info(fmt.Sprintf("appended linked record %d to %d", entry.ParRef, entryID))
		linkedRecordsInfo = append(linkedRecordsInfo,
			LinkedRecordInfo{RefEntry: uint32(entry.ParRef), StartVCN: entry.StartVcn, RefSeq: entry.ParSeq})
	}
	return linkedRecordsInfo
}

func (attrListEntries AttributeListEntries) FindType() string {
	return attrListEntries.Header.GetType()
}

func (attrListEntries AttributeListEntries) IsNoNResident() bool {
	return attrListEntries.Header.IsNoNResident()
}

func (attrListEntries AttributeListEntries) GetInfo() string {
	var txt strings.Builder
	fmt.Fprintf(&txt, "%s", attrListEntries.Header.GetInfo())

	for _, attrList := range attrListEntries.Entries {
		fmt.Fprintf(&txt, "Attr List Type %s MFT Ref %d startVCN %d name %s \n",
			attrList.GetType(), attrList.ParRef, attrList.StartVcn, attrList.Name)
	}
	return txt.String()
}
