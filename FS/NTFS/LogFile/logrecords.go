package logfile

import (
	"fmt"

	"github.com/aarsakian/FileSystemForensics/utils"
)

var logRecordFlagsMap = map[uint16]string{
	0x0001: "MULTI_PAGE",   // Record spans multiple pages
	0x0002: "LAST_IN_TXN",  // Last record in a transaction
	0x0004: "NO_REDO",      // Redo not required
	0x0008: "NO_UNDO",      // Undo not required
	0x0010: "COMPENSATION", // Compensation Log Record (CLR)
	0x0020: "PINNED",       // Record is pinned in memory
	0x0040: "DIRTY",        // Record modifies metadata
	0x0080: "REDO_ONLY",    // Redo-only operation
	0x0100: "UNDO_ONLY",    // Undo-only operation
	0x0200: "REDO_UNDO",    // Both redo and undo present
}

var logRecordTypeMap = map[uint16]string{
	0x00: "NOOP",
	0x01: "COMPENSATION_LOG_RECORD",
	0x02: "INITIALIZE_FILE_RECORD_SEGMENT",
	0x03: "DEALLOCATE_FILE_RECORD_SEGMENT",
	0x04: "WRITE_END_OF_FILE_RECORD_SEGMENT",
	0x05: "CREATE_ATTRIBUTE",
	0x06: "DELETE_ATTRIBUTE",
	0x07: "UPDATE_RESIDENT_VALUE",
	0x08: "UPDATE_NONRESIDENT_VALUE",
	0x09: "UPDATE_MAPPING_PAIRS",
	0x0A: "DELETE_DIRTY_CLUSTERS",
	0x0B: "SET_NEW_ATTRIBUTE_SIZES",
	0x0C: "ADD_INDEX_ENTRY_ROOT",
	0x0D: "DELETE_INDEX_ENTRY_ROOT",
	0x0E: "ADD_INDEX_ENTRY_ALLOCATION",
	0x0F: "DELETE_INDEX_ENTRY_ALLOCATION",
	0x10: "WRITE_END_OF_INDEX_BUFFER",
	0x11: "SET_INDEX_ENTRY_VCN_ROOT",
	0x12: "SET_INDEX_ENTRY_VCN_ALLOCATION",
	0x13: "UPDATE_FILENAME_ROOT",
	0x14: "UPDATE_FILENAME_ALLOCATION",
	0x15: "SET_BITS_IN_NONRESIDENT_BITMAP",
	0x16: "CLEAR_BITS_IN_NONRESIDENT_BITMAP",
}

// 32 bytes
// offsetbits = 64 -seqnumbits
// LSN &((1<<offsetBits)-1)
// offset * 8 = recordoffset from the start of $LogFile sequence number + offset (Sequence number=SeqNumBits from restart area header)
type LogRecordHeader struct {
	LSN              uint64 // upper bits defined by SeqNumBits, lower bits =offset
	PreviousLSN      uint64 // 0x0C: Link to previous log record,
	UndoNextLSN      uint64 // 0x14: For rollback operations
	ClientDataLength uint32
	ClientSeqNumber  uint16
	ClientID         uint16 // 0x1C: Identifies the client (e.g., NTFS)
	RecordType       uint32 // 1 = Transaction record 2 = Checkpoint record
	TranscationID    uint32
	Flags            uint16 // 1 = extents to next log log record page
	Reserved         [6]byte
}

type LogRecord struct {
	RedoOperation         uint16 // 0x24: Operation code for redo
	UndoOperation         uint16 // 0x26: Operation code for undo
	RedoOffset            uint16 // 0x28: Offset to redo data
	RedoLength            uint16 // 0x2A: Length of redo data
	UndoOffset            uint16 // 0x2C: Offset to undo data >0x28
	UndoLength            uint16 // 0x2E: Length of undo data
	TargetAttributeOffset uint16 //
	LCNsTOFollow          uint16
	RecordOffset          uint16
	AttributeOffset       uint16
	MFTClusterIndex       uint16
	Reserved1             [2]byte
	TargetVCN             uint32
	Reserved2             [4]byte
	TargetLCN             uint32
	Reserved3             [4]byte
	// Followed by variable-length redo/undo data
}

func (logRecordHeader *LogRecordHeader) Parse(data []byte) {
	utils.Unmarshal(data, logRecordHeader)

}

func (logRecord *LogRecord) Parse(data []byte) {
	utils.Unmarshal(data, logRecord)
}

func (logRecordHeader LogRecordHeader) GetOffset(seqnumbits uint32) uint64 {
	offsetBits := 64 - seqnumbits
	return logRecordHeader.LSN & ((1 << offsetBits) - 1)
}

func DecodeLogRecordType(code uint16) string {
	if name, ok := logRecordTypeMap[code]; ok {
		return name
	}
	return fmt.Sprintf("UNKNOWN_TYPE_0x%02X", code)
}

func DecodeLogRecordFlags(flags uint16) []string {
	var decoded []string
	for bit, name := range logRecordFlagsMap {
		if flags&bit != 0 {
			decoded = append(decoded, name)
		}
	}
	return decoded
}
