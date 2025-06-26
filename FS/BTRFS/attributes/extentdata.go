package attributes

import (
	"fmt"

	"github.com/aarsakian/FileSystemForensics/utils"
)

type ExtentData struct {
	Generation     uint64
	LogicalDataLen uint64 //size of decoded text
	Compression    uint8
	Encryption     uint8
	OtherEncoding  uint16
	Type           uint8
	ExtentDataRem  *ExtentDataRem
	InlineData     []byte
}

// used when no compression, encryption other encoding is used non inline
type ExtentDataRem struct {
	LogicaAddress uint64 //logical address of extent
	Size          uint64
	Offset        uint64
	LogicalBytes  uint64
}

func (extentData *ExtentData) Parse(data []byte) int {
	offset, _ := utils.Unmarshal(data, extentData)
	if extentData.Compression == 0 && extentData.Encryption == 0 &&
		ExtentTypes[extentData.Type] != "Inline Extent" {
		extentData.ExtentDataRem = new(ExtentDataRem)
		curOffset, _ := utils.Unmarshal(data[offset:], extentData.ExtentDataRem)
		offset += curOffset

	} else if extentData.Compression == 0 && extentData.Encryption == 0 &&
		ExtentTypes[extentData.Type] == "Inline Extent" {
		copy(extentData.InlineData, data[offset:])
		offset = len(data)
	}
	return offset
}

func (extentData ExtentData) ShowInfo() {
	fmt.Printf("%s \n", extentData.GetInfo())
}

func (extentData ExtentData) GetType() string {
	return ExtentTypes[extentData.Type]
}

func (extentData ExtentData) GetInfo() string {
	return fmt.Sprintf("%s %d", extentData.GetType(), extentData.LogicalDataLen)
}
