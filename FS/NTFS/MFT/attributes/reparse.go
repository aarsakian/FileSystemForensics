package attributes

import (
	"fmt"

	"github.com/aarsakian/FileSystemForensics/logger"
	"github.com/aarsakian/FileSystemForensics/utils"
)

/*
31  Microsoft bit
30  Reserved bit zero for non microsoft tags
29  Name Surrogate bit, bit is set to 1 file/directory represents another entity
28  Directory bit can have children
28-16 reserverd zero
16-0 uint16 uniquely identifies the owner of the reparse
*/
var ReparseTagflags = map[uint32]string{0xa000003: "MountPoint", 0x80000013: "dedup", 0xa00000c: "Symbolink",
	0x80000016: "dynamic file filter", 0x80000017: "Windows Overlay filter", 0x9000001a: "cloud", 0x9000101a: "cloud 1",
	0x9000201a: "cloud 2", 0x9000301a: "cloud 3", 0x9000401a: "cloud 4", 0x9000501a: "cloud 5", 0x9000601a: "cloud 6",
	0x9000701a: "cloud 7", 0x9000801a: "cloud 8", 0x9000901a: "cloud 9", 0x9000a01a: "cloud A", 0x9000b01a: "cloud B", 0x9000c01a: "cloud C",
	0x9000d01a: "cloud D", 0x9000e01a: "cloud E", 0x9000f01a: "cloud F"}

var SymbolikReparseFlags = map[int]string{0: "FullPathName", 1: "RelativePath"}

type Reparse struct {
	Flags     [4]byte
	Size      uint16
	Unused    [2]byte
	Data      []byte
	Header    *AttributeHeader
	Symbolink *SymbolikReparse
}

type SymbolikReparse struct {
	SubstituteNameOffset uint16
	SubstituteNameLen    uint16
	PrintNameOffset      int16
	PrintNameLen         uint16
	Flags                [4]byte
	Header               *AttributeHeader
	Name                 string
	PrintName            string
}

func (reparse *Reparse) SetHeader(header *AttributeHeader) {
	reparse.Header = header
}

func (reparse Reparse) GetHeader() AttributeHeader {
	return *reparse.Header
}

func (reparse *Reparse) Parse(data []byte) {
	readOffset, _ := utils.Unmarshal(data, reparse)
	if len(data) < readOffset {
		logger.FSLogger.Warning(fmt.Sprintf("Reparse data not enough %d", len(data)))
		return
	}
	if reparse.GetFlagInfo() == "Symbolink" {
		symbolink := new(SymbolikReparse)
		symbolink.Parse(data[readOffset:])

		reparse.Symbolink = symbolink

	}

}

func (symbolinkReparse *SymbolikReparse) Parse(data []byte) {
	utils.Unmarshal(data, symbolinkReparse)

	symbolinkReparse.Name = utils.DecodeUTF16(data[uint16(symbolinkReparse.SubstituteNameOffset) : uint16(symbolinkReparse.SubstituteNameOffset)+
		symbolinkReparse.SubstituteNameLen])

	symbolinkReparse.PrintName = utils.DecodeUTF16(data[uint16(symbolinkReparse.PrintNameOffset) : 16+
		uint16(symbolinkReparse.PrintNameLen)])
}

func (reparse Reparse) IsNoNResident() bool {
	return reparse.Header.IsNoNResident()
}

func (reparse Reparse) FindType() string {
	return reparse.Header.GetType()
}

func (reparse Reparse) GetFlagInfo() string {
	return ReparseTagflags[utils.ToUint32(reparse.Flags[:])]
}

func (reparse Reparse) ShowInfo() {
	fmt.Printf("Type %s flag %s\n", reparse.FindType(), reparse.GetFlagInfo())
	if reparse.Symbolink != nil {
		reparse.Symbolink.ShowInfo()
	}
}

func (symbolinkReparse SymbolikReparse) ShowInfo() {
	fmt.Printf("Target Name %s Print Name %s\n", symbolinkReparse.Name, symbolinkReparse.PrintName)
}
