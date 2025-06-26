package attributes

import (
	"fmt"

	"github.com/aarsakian/FileSystemForensics/logger"
	"github.com/aarsakian/FileSystemForensics/utils"
)

// Dir Index items always contain one entry and their key
// is the index of the file in the current directory
type DirIndex struct {
	Transid uint64
	DataLen uint16
	NameLen uint16
	Type    uint8
	Name    string
}

func (dirIndex *DirIndex) Parse(data []byte) int {
	offset, _ := utils.Unmarshal(data, dirIndex)
	if offset+int(dirIndex.NameLen) < len(data) {
		dirIndex.Name = string(data[offset : offset+int(dirIndex.NameLen)]) //?
		return offset + int(dirIndex.NameLen)
	} else {
		logger.FSLogger.Warning(fmt.Sprintf("DirIdx nameLen %d exceeding available data %d", dirIndex.NameLen, len(data)))
		return len(data)
	}
}

func (dirIndex DirIndex) GetType() string {
	return DirTypes[dirIndex.Type]
}

func (dirIndex DirIndex) GetInfo() string {
	return fmt.Sprintf("transid  %d  type %s name %s", dirIndex.Transid, dirIndex.Name, dirIndex.GetType())
}
func (dirIndex DirIndex) ShowInfo() {
	fmt.Printf("%s \n", dirIndex.GetInfo())
}
