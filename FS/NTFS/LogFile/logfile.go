package logfile

import (
	"bytes"

	"github.com/aarsakian/FileSystemForensics/utils"
)

type LogFile struct {
	RCRDRecords RCRDRecords
	RSTRRecords RSTRRecords
}

type ClientRecord struct {
	OldestLSN        uint64
	LSNClientRestart uint64
	PrevClient       uint16
	NextClient       uint16
	SeqNum           uint16
	ClientNameLen    uint16
	ClientNameOffset uint16
	Flags            uint16
	ClientName       string
}

type LogRecordHeader struct {
	OldestLSN        uint64
	LSNClientRestart uint64
	PrevLSN          uint64
	UndoNextLSN      uint64
	SeqNum           uint16
	Reserved         [6]byte
	ClientNameLen    uint32
	ClientName       string
}

type UpdateSequenceArray struct { //USA
	SequenceNumber  uint16
	ReplacementData []uint16
}

func (logfile *LogFile) Parse(data []byte) {

	offset := 0
	for offset < len(data) {
		if bytes.Equal(data[offset:offset+4], []byte{0x52, 0x53, 0x54, 0x52}) {
			rcrs := new(RSTR)
			utils.Unmarshal(data[offset:], rcrs)

			err := rcrs.ProcessFixUpArrays(data[offset:])
			if err == nil {
				rcrs.ReplaceFixupValues(data[offset:])

				restartArea := new(RestartAreaHeader)
				restartArea.Parse(data[offset+int(rcrs.UpdateFixUpArrOffset)+int(rcrs.UpdateFixUpArrSize*2):])

				logfile.RSTRRecords = append(logfile.RSTRRecords, *rcrs)
			}
		} else if bytes.Equal(data[offset:offset+4], []byte{0x52, 0x43, 0x52, 0x44}) {
			rcrd := new(RCRD)
			utils.Unmarshal(data[offset:], rcrd)

			err := rcrd.ProcessFixUpArrays(data[offset:])
			if err == nil {
				rcrd.ReplaceFixupValues(data[offset:])

				//is enough for the header
				if 4096-rcrd.NextRecordOffset >= 48 {
					logRecordHeader := new(LogRecordHeader)
					logRecordHeader.Parse(data[offset+int(rcrd.NextRecordOffset):])

					rcrd.LogRecordHeaders = append(rcrd.LogRecordHeaders, *logRecordHeader)

				}
				logfile.RCRDRecords = append(logfile.RCRDRecords, *rcrd)

			}

		}
		offset += 4096 // page size
	}

}

func (logRecordHeader *LogRecordHeader) Parse(data []byte) {
	utils.Unmarshal(data, logRecordHeader)

}

/*func (logRecordHeader LogRecordHeader) IsValid() bool {
	return logRecordHeader.RecordType <= 22 && logRecordHeader.LSN != 0
}*/

func (restartArea *RestartAreaHeader) Parse(data []byte) {
	utils.Unmarshal(data, restartArea)

	for curClient := 0; curClient < int(restartArea.LogClientCount); curClient++ {
		clientRecord := new(ClientRecord)
		utils.Unmarshal(data[restartArea.ClientArrayOffset:], clientRecord)
	}

}
