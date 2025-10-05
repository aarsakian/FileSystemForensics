package attributes

import (
	"fmt"

	"github.com/aarsakian/FileSystemForensics/utils"
)

var FileAttributes = map[uint32]string{
	0x00000001: "READONLY",
	0x00000002: "HIDDEN",
	0x00000004: "SYSTEM",
	0x00000010: "DIRECTORY",
	0x00000020: "ARCHIVE",
	0x00000040: "DEVICE", // Reserved
	0x00000080: "NORMAL",
	0x00000100: "TEMPORARY",
	0x00000200: "SPARSE_FILE",
	0x00000400: "REPARSE_POINT",
	0x00000800: "COMPRESSED",
	0x00001000: "OFFLINE",
	0x00002000: "NOT_CONTENT_INDEXED",
	0x00004000: "ENCRYPTED",
	0x00008000: "INTEGRITY_STREAM",
	0x00010000: "VIRTUAL",
	0x00020000: "NO_SCRUB_DATA",
	0x00040000: "EA", // Also used for RECALL_ON_OPEN
	0x00080000: "PINNED",
	0x00100000: "UNPINNED",
	0x00400000: "RECALL_ON_DATA_ACCESS",
}

type SIAttribute struct {
	Crtime         utils.WindowsTime
	Mtime          utils.WindowsTime
	MFTmtime       utils.WindowsTime
	Atime          utils.WindowsTime
	FileAttributes uint32
	Maxver         uint32
	Ver            uint32
	ClassID        uint32
	OwnID          uint32
	SecID          uint32
	Quota          uint64
	USN            uint64 //most recent UsnJrnl record
	Header         *AttributeHeader
}

func (siattr *SIAttribute) SetHeader(header *AttributeHeader) {
	siattr.Header = header
}

func (siattr SIAttribute) GetHeader() AttributeHeader {
	return *siattr.Header
}

func (siattr *SIAttribute) Parse(data []byte) {
	utils.Unmarshal(data, siattr)
}

func (siattr SIAttribute) FindType() string {
	return siattr.Header.GetType()
}

func (siattr SIAttribute) IsNoNResident() bool {
	return siattr.Header.IsNoNResident() // always resident
}

func (siattr SIAttribute) GetTimestamps() (string, string, string, string) {
	atime := siattr.Atime.ConvertToIsoTime()
	ctime := siattr.Crtime.ConvertToIsoTime()
	mtime := siattr.Mtime.ConvertToIsoTime()
	mftime := siattr.MFTmtime.ConvertToIsoTime()
	return atime, ctime, mtime, mftime
}

func (siattr SIAttribute) ShowInfo() {
	atime, ctime, mtime, mfttime := siattr.GetTimestamps()

	fmt.Printf(" %s usn  %d atime %s ctime %s mtime %s mfttime %s file attr %s\n",
		siattr.FindType(), siattr.USN, atime, ctime, mtime, mfttime,
		siattr.GetFileAttributes())
}

func (siattr SIAttribute) GetFileAttributes() string {
	var flags string
	for bit, name := range FileAttributes {
		if siattr.FileAttributes&bit != 0 {

			flags += " " + name

		}
	}
	return flags
}
