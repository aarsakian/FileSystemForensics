package attributes

import (
	"fmt"
	"strings"
)

type BitMap struct {
	AllocationStatus []byte
	Header           *AttributeHeader
}

func (bitmap *BitMap) SetHeader(header *AttributeHeader) {
	bitmap.Header = header
}

func (bitmap BitMap) GetHeader() AttributeHeader {
	return *bitmap.Header
}

func (bitmap *BitMap) Parse(data []byte) {
	copy(bitmap.AllocationStatus, data)
}

func (bitmap BitMap) FindType() string {
	return bitmap.Header.GetType()
}

func (bitmap BitMap) IsNoNResident() bool {
	return bitmap.Header.IsNoNResident()
}

func (bitmap BitMap) GetInfo() string {
	var txt strings.Builder
	txt.WriteString(fmt.Sprintf("type %s \n", bitmap.FindType()))
	pos := 1
	for _, byteval := range bitmap.AllocationStatus {
		bitmask := uint8(0x01)
		shifter := 0
		for bitmask < 128 {

			bitmask = 1 << shifter
			txt.WriteString(fmt.Sprintf("cluster/entry  %d status %d \t", pos, byteval&bitmask))
			pos++
			shifter++
		}

	}
	return txt.String()
}
