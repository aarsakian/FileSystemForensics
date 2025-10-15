package logfile

/*
Restart Area (2) | Buffer Area (2 for LFS 1.1) (32 for LFS 2.0) | Norma Logging Area
*/

import (
	"bytes"

	"github.com/aarsakian/FileSystemForensics/utils"
)

const PAGE_SIZE = 4096

type LogFile struct {
	Page_Size   int
	Version     string
	RCRDRecords RCRDRecords
	RSTRRecords RSTRRecords
}

type UpdateSequenceArray struct { //USA
	SequenceNumber  uint16
	ReplacementData []uint16
}

func (logfile *LogFile) Parse(data []byte) error {
	offset := 0
	logfile.RSTRRecords = make(RSTRRecords, 2)
	for idx := range logfile.RSTRRecords {
		rcrs := new(RSTR)
		err := rcrs.Parse(data[offset:])
		if err != nil {
			return err
		}

		logfile.RSTRRecords[idx] = *rcrs

		logfile.Page_Size = int(rcrs.LogPageSize)
		logfile.Version = rcrs.GetVersion()

		offset += PAGE_SIZE
	}

outer:
	for offset < len(data) {
		switch logfile.Version {
		case "2.0":
			offset += 32 * logfile.Page_Size
		case "1.1":
			offset += 2 * logfile.Page_Size
		default:
			break outer
		}

		if bytes.Equal(data[offset:offset+4], []byte{0x52, 0x43, 0x52, 0x44}) {
			rcrd := new(RCRD)
			utils.Unmarshal(data[offset:], rcrd)
			err := rcrd.Parse(data[offset:])
			if err == nil {
				logfile.RCRDRecords = append(logfile.RCRDRecords, *rcrd)
			}

		}

		offset += int(logfile.Page_Size)
	}
	return nil
}

/*func (logRecordHeader LogRecordHeader) IsValid() bool {
	return logRecordHeader.RecordType <= 22 && logRecordHeader.LSN != 0
}*/
