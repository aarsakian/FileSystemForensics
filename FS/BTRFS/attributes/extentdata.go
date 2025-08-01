package attributes

import (
	"fmt"

	"github.com/aarsakian/FileSystemForensics/utils"
)

type ExtentData struct {
	Generation    uint64
	DecodedSize   uint64 //size of decoded text after decompression
	Compression   uint8
	Encryption    uint8
	OtherEncoding uint16
	Type          uint8
	ExtentDataRem *ExtentDataRem
	InlineData    []byte
}

// used when no compression, encryption other encoding is used non inline
type ExtentDataRem struct {
	LogicalAddress uint64 //logical address of extent
	Psize          uint64 //physical size of the extent on disk
	Offset         uint64 //offset within extent
	LSize          uint64 //logical size of the extent
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

func (extentData ExtentData) GetCompressionType() string {
	return CompressionTypes[int(extentData.Compression)]
}

func (extentData ExtentData) GetInfo() string {
	if extentData.ExtentDataRem != nil {
		return fmt.Sprintf("%s %s %s", extentData.GetType(), extentData.GetCompressionType(), extentData.ExtentDataRem.GetInfo())
	} else {
		return fmt.Sprintf("%s %s", extentData.GetType(), extentData.GetCompressionType())
	}

}

func (extentDataRem ExtentDataRem) GetInfo() string {
	return fmt.Sprintf("Logical Offset %d logical size %d extent offset %d",
		extentDataRem.LogicalAddress, extentDataRem.LSize, extentDataRem.Offset)
}
