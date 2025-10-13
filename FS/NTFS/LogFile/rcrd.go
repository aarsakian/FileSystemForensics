package logfile

import (
	"errors"
	"fmt"

	"github.com/aarsakian/FileSystemForensics/logger"
	"github.com/aarsakian/FileSystemForensics/utils"
)

var flagNameMap = map[uint32]string{
	0x00000001: "VALID",
	0x00000002: "DIRTY",
	0x00000004: "END",
	0x00000008: "MULTI_SECTOR_HEADER",
}

type RCRDRecords []RCRD

// 64B size
type RCRD struct {
	Signature            [4]byte
	UpdateFixUpArrOffset uint16 //4-5      offset values are relative to the start of the entry.
	UpdateFixUpArrSize   uint16 //6-7
	LastLSNORFileOffset  uint64 //
	Flags                uint32
	PageCount            uint16
	PagePosition         uint16
	NextRecordOffset     uint16 //26 offset to the free space of the page
	Reserved1            [6]byte
	LastEndLSN           uint64 // last LSN of the page 0x28 offset+
	LogRecordHeaders     []LogRecordHeader
	FixUp                *utils.FixUp // 2x9
}

func (rcrd *RCRD) ProcessFixUpArrays(data []byte) error {
	if len(data) < int(2*rcrd.UpdateFixUpArrSize) {
		msg := fmt.Sprintf("Data not enough to parse fixup array by %d", int(2*rcrd.UpdateFixUpArrSize)-len(data))
		logger.FSLogger.Warning(msg)
		return errors.New(msg)
	}
	fixuparray := data[rcrd.UpdateFixUpArrOffset : rcrd.UpdateFixUpArrOffset+2*rcrd.UpdateFixUpArrSize]
	var fixupvals [][]byte
	val := 2
	for val < len(fixuparray) {

		fixupvals = append(fixupvals, fixuparray[val:val+2])
		val += 2
	}
	//2bytes for USN update Sequence Number, rest is USA Update Sequence Array 4 byte
	if len(fixuparray) > 2 {
		rcrd.FixUp = &utils.FixUp{Signature: fixuparray[:2], OriginalValues: fixupvals}
		return nil
	} else {
		msg := fmt.Sprintf("fixup array len smaller than 2 %d", len(fixuparray))
		logger.FSLogger.Warning(msg)
		return errors.New(msg)
	}

}

func (rcrd RCRD) ReplaceFixupValues(data []byte) {
	for idx := 1; idx < int(rcrd.UpdateFixUpArrSize); idx++ {
		//first is the fixup itself
		if rcrd.FixUp.Signature[0] == data[(idx-1)*512+510] &&
			rcrd.FixUp.Signature[1] == data[(idx-1)*512+511] {
			//4096 size therefore every 512 bytes need 8 passes
			data[(idx-1)*512+510] = rcrd.FixUp.OriginalValues[idx-1][0]
			data[(idx-1)*512+511] = rcrd.FixUp.OriginalValues[idx-1][1]
		} else {
			break
		}

	}
}

func (rcrd RCRD) DecodeFlags() []string {
	decoded := []string{}
	for bit, name := range flagNameMap {
		if rcrd.Flags&bit != 0 {
			decoded = append(decoded, name)
		}
	}
	return decoded
}
