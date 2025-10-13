package logfile

import (
	"bytes"

	"github.com/aarsakian/FileSystemForensics/utils"
)

type LogFile struct {
	RCRDRecords RCRDRecords
	RSTRRecords RSTRRecords
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

				restartAreaOffset := offset + int(rcrs.UpdateFixUpArrOffset) + int(rcrs.UpdateFixUpArrSize*2)

				clientRecord := new(ClientRecord)
				clientRecord.Parse(data[restartAreaOffset+
					int(restartArea.ClientArrayOffset) : restartAreaOffset+int(restartArea.ClientArrayOffset)+
					int(restartArea.ClientArrayLength)])

				restartArea.ClientRecords = append(restartArea.ClientRecords, *clientRecord)
				logfile.RSTRRecords = append(logfile.RSTRRecords, *rcrs)
			}
		} else if bytes.Equal(data[offset:offset+4], []byte{0x52, 0x43, 0x52, 0x44}) {
			rcrd := new(RCRD)
			utils.Unmarshal(data[offset:], rcrd)

			err := rcrd.ProcessFixUpArrays(data[offset:])
			if err == nil {
				rcrd.ReplaceFixupValues(data[offset:])

				//is enough for the header

				logRecordHeader := new(LogRecordHeader)
				logRecordHeader.Parse(data[offset+64:])

				logRecord := new(LogRecord)
				logRecord.Parse(data[offset+64+32 : offset+64+32+int(logRecordHeader.DataLength)])

				rcrd.LogRecordHeaders = append(rcrd.LogRecordHeaders, *logRecordHeader)

				logfile.RCRDRecords = append(logfile.RCRDRecords, *rcrd)

			}

		}
		offset += 4096 // page size
	}

}

/*func (logRecordHeader LogRecordHeader) IsValid() bool {
	return logRecordHeader.RecordType <= 22 && logRecordHeader.LSN != 0
}*/

func (restartArea *RestartAreaHeader) Parse(data []byte) {
	utils.Unmarshal(data, restartArea)

	for curClient := 0; curClient < int(restartArea.LogClientCount); curClient++ {
		clientRecord := new(LogRecordHeader)
		utils.Unmarshal(data[restartArea.ClientArrayOffset:], clientRecord)
	}

}
