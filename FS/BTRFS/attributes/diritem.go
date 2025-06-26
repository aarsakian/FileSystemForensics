package attributes

import (
	"fmt"

	"github.com/aarsakian/FileSystemForensics/logger"
	"github.com/aarsakian/FileSystemForensics/utils"
)

// allow lookup by name
type DirItem struct {
	Transid   uint64
	DataLen   uint16 //Xattr Len 0 otherwise
	NameLen   uint16 //Xattr Name len or dir len
	Type      uint8
	Name      string //name of ordinary dir entry or xattr name
	XattValue string
}

func (dirItem *DirItem) Parse(data []byte) int {
	offset, _ := utils.Unmarshal(data, dirItem)

	if offset+int(dirItem.NameLen) < len(data) { // needs check
		dirItem.Name = string(data[offset : offset+int(dirItem.NameLen)])
		if dirItem.DataLen > 1 {
			dirItem.XattValue = string(data[offset+int(dirItem.NameLen) : offset+int(dirItem.NameLen)+int(dirItem.DataLen)])
			return offset + int(dirItem.NameLen) + int(dirItem.DataLen)
		} else {
			return offset + int(dirItem.NameLen)
		}

	} else {
		logger.FSLogger.Warning(fmt.Sprintf("Dir Item nameLen %d exceeding available data %d", dirItem.NameLen, len(data)))
		return len(data)
	}

}

func (dirItem DirItem) GetType() string {
	return DirTypes[dirItem.Type]
}

func (dirItem DirItem) ShowInfo() {
	fmt.Printf("%s \n", dirItem.GetInfo())
}

func (dirItem DirItem) GetInfo() string {
	return fmt.Sprintf("transid  %d  type %s name %s xattr vAL %s", dirItem.Transid, dirItem.GetType(), dirItem.Name, dirItem.XattValue)
}

/*	dataItem := node.LeafNode.DataItems[idx].(*leafnode.DirItem)
	fileDirEntry.Flags = dataItem.GetType()
	fileDirEntry.Type = dataItem.GetType()*/
