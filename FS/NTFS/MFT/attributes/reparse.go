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
var ReparseTagflags = map[string]string{"a000003": "MountPoint", "80000013": "dedup", "a00000c": "Symbolink",
	"80000016": "dynamic file filter", "80000017": "Windows Overlay filter", "9000001a": "cloud", "9000101a": "cloud 1",
	"9000201a": "cloud 2", "9000301a": "cloud 3", "9000401a": "cloud 4", "9000501a": "cloud 5", "9000601a": "cloud 6",
	"9000701a": "cloud 7", "9000801a": "cloud 8", "9000901a": "cloud 9", "9000a01a": "cloud A", "9000b01a": "cloud B", "9000c01a": "cloud C",
	"9000d01a": "cloud D", "9000e01a": "cloud E", "9000f01a": "cloud F"}

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
		logger.MFTExtractorlogger.Warning(fmt.Sprintf("Reparse data not enough %d", len(data)))
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
	return ReparseTagflags[utils.Hexify(utils.Bytereverse(reparse.Flags[:]))]
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
