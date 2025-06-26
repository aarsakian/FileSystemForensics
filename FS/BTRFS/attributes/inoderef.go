package attributes

import (
	"fmt"

	"github.com/aarsakian/FileSystemForensics/utils"
)

// name entry for inode
type InodeRef struct {
	Index  uint64
	Length uint16
	Name   string
}

func (inodeRef *InodeRef) Parse(data []byte) int {
	offset, _ := utils.Unmarshal(data, inodeRef)
	inodeRef.Name = string(data[offset : offset+int(inodeRef.Length)])
	return offset + int(inodeRef.Length)
}

func (inodeRef InodeRef) ShowInfo() {
	fmt.Printf("%s \n", inodeRef.GetInfo())
}

func (inodeRef InodeRef) GetInfo() string {
	return fmt.Sprintf("idx %d %s ", inodeRef.Index, inodeRef.Name)
}

/*fileDirEntry.Name = dataItem.Name
fileDirEntry.Index = int(dataItem.Index)*/
