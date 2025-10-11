package logfile

import (
	"errors"
	"fmt"

	"github.com/aarsakian/FileSystemForensics/logger"
	"github.com/aarsakian/FileSystemForensics/utils"
)

type RSTRRecords []RSTR

type RSTR struct {
	Signature            [4]byte
	UpdateFixUpArrOffset uint16 //4-5      offset values are relative to the start of the entry.
	UpdateFixUpArrSize   uint16 //6-7
	CheckDiskLSN         uint64
	SystemPageSize       uint32
	LogPageSize          uint32
	RestartAreaOffset    uint16
	MinorVersion         uint16
	MajorVersion         uint16
	FixUp                *utils.FixUp
}

type RestartAreaHeader struct {
	CurrentLSN          uint64
	LogClientCount      uint16
	ClientFreeList      uint16
	ClientInUseList     uint16
	Flags               uint16
	SeqNumberBits       uint32
	ClientArrayLength   uint16
	ClientArrayOffset   uint16
	FileSize            uint64
	LastLSNDataLen      uint32
	LogRecordHDLen      uint16
	LogPageDataOffset   uint16
	RestartLogOpenCount uint32
	ClientRecords       []ClientRecord
}

type ClientRecord struct {
	OldestLSN        uint64 // 0x0C: Oldest LSN still needed by this client
	RestartLSN       uint64 // 0x14: LSN used for restart recovery
	PrevClient       uint16 // 0x04: Offset to the next client record in the client list
	NextClient       uint16 // 0x08: Offset to the previous client record
	SeqNumber        uint16
	ALign1           uint16 // 0x1A: Padding or alignment
	Align2           uint32 // 0x20: Padding or alignment
	ClientNameLength uint32 // 0x1C: Offset to the client name string

	// Followed by variable-length client name and client-specific data
	ClientName string
}

func (rcrs *RSTR) ProcessFixUpArrays(data []byte) error {
	if len(data) < int(2*rcrs.UpdateFixUpArrSize) {
		msg := fmt.Sprintf("Data not enough to parse fixup array by %d", int(2*rcrs.UpdateFixUpArrSize)-len(data))
		logger.FSLogger.Warning(msg)
		return errors.New(msg)
	}
	fixuparray := data[rcrs.UpdateFixUpArrOffset : rcrs.UpdateFixUpArrOffset+2*rcrs.UpdateFixUpArrSize]
	var fixupvals [][]byte
	val := 2
	for val < len(fixuparray) {

		fixupvals = append(fixupvals, fixuparray[val:val+2])
		val += 2
	}
	//2bytes for USN update Sequence Number, rest is USA Update Sequence Array 4 byte
	if len(fixuparray) > 2 {
		rcrs.FixUp = &utils.FixUp{Signature: fixuparray[:2], OriginalValues: fixupvals}
		return nil
	} else {
		msg := fmt.Sprintf("fixup array len smaller than 2 %d", len(fixuparray))
		logger.FSLogger.Warning(msg)
		return errors.New(msg)
	}

}

func (rcrs RSTR) ReplaceFixupValues(data []byte) {
	for idx := 1; idx < int(rcrs.UpdateFixUpArrSize); idx++ {
		//first is the fixup itself
		if rcrs.FixUp.Signature[0] == data[(idx-1)*512+510] &&
			rcrs.FixUp.Signature[1] == data[(idx-1)*512+511] {
			//4096 size therefore every 512 bytes need 8 passes
			data[(idx-1)*512+510] = rcrs.FixUp.OriginalValues[idx-1][0]
			data[(idx-1)*512+511] = rcrs.FixUp.OriginalValues[idx-1][1]
		} else {
			break
		}

	}
}

func (ClientRecord *ClientRecord) Parse(data []byte) {
	utils.Unmarshal(data, ClientRecord)
	ClientRecord.ClientName = utils.DecodeUTF16(data[32 : 32+
		uint32(ClientRecord.ClientNameLength)])

}
