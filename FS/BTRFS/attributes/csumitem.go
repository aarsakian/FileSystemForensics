package attributes

import "github.com/aarsakian/FileSystemForensics/utils"

// every 4kb of every file written contains checksum
type CsumItem struct {
	Checksums []uint32 //crc32
}

func (checkSumItem *CsumItem) Parse(data []byte) int {
	offset := 0

	for offset < len(data) {
		checkSumItem.Checksums = append(checkSumItem.Checksums, utils.ToUint32(data[offset:offset+4]))
		offset += 4
	}
	return offset
}

func (checkSumItem CsumItem) ShowInfo() {

}

func (checkSumItem CsumItem) GetInfo() string {
	return ""
}
