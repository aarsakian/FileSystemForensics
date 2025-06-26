package attributes

import "github.com/aarsakian/FileSystemForensics/utils"

// size 18
// used to identify snapshot or volume
type RootRef struct {
	DirID  uint64
	Index  uint64
	Length uint16
	Name   string
}

// 26
type RootBackRef struct {
	DirID  uint64
	Index  uint64
	Length uint16
	Name   string
}

func (rootRef *RootRef) Parse(data []byte) int {
	offset, _ := utils.Unmarshal(data, rootRef)
	if int(rootRef.Length) < offset { //error in parsing?
		rootRef.Name = string(data[offset : offset+int(rootRef.Length)])
		return offset + int(rootRef.Length)
	} else {
		return offset
	}

}

func (rootRef RootRef) ShowInfo() {

}

func (rootRef RootRef) GetInfo() string {
	return ""
}

func (rootBackRef *RootBackRef) Parse(data []byte) int {
	offset, _ := utils.Unmarshal(data, rootBackRef)
	if int(rootBackRef.Length) < offset { //error in parsing?
		rootBackRef.Name = string(data[offset : offset+int(rootBackRef.Length)])
		return offset + int(rootBackRef.Length)
	} else {
		return offset
	}
}

func (rootBackRef RootBackRef) GetInfo() string {
	return ""
}

func (rootBackRef RootBackRef) ShowInfo() {

}
