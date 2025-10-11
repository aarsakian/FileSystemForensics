package logfile

import "fmt"

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
	0x0A: "UPDATE_FILE_NAME",
	0x0B: "SET_NEW_PARENT_DIRECTORY",
	0x0C: "UPDATE_SECURITY_DESCRIPTOR",
	0x0D: "UPDATE_INDEX_ROOT",
	0x0E: "UPDATE_INDEX_ALLOCATION",
	0x0F: "UPDATE_BITMAP",
	0x10: "UPDATE_VOLUME_BITMAP",
	0x11: "UPDATE_BOOT_SECTOR",
	0x12: "UPDATE_LOG_FILE",
}

type LogRecordHeader struct {
	RecordType       uint16 // 0x00: Type of log record (e.g., 0x01 = CLR)
	ClientSeqNumber  uint16 // 0x02: Sequence number for the client
	LSN              uint64 // 0x04: Log Sequence Number (8 bytes)
	PreviousLSN      uint64 // 0x0C: Link to previous log record
	UndoNextLSN      uint64 // 0x14: For rollback operations
	ClientID         uint32 // 0x1C: Identifies the client (e.g., NTFS)
	RecordLength     uint16 // 0x20: Total length of this log record
	Flags            uint16 // 0x22: Bitmask (e.g., dirty, committed)
	RedoOperation    uint16 // 0x24: Operation code for redo
	UndoOperation    uint16 // 0x26: Operation code for undo
	RedoOffset       uint16 // 0x28: Offset to redo data
	RedoLength       uint16 // 0x2A: Length of redo data
	UndoOffset       uint16 // 0x2C: Offset to undo data
	UndoLength       uint16 // 0x2E: Length of undo data
	TargetVCN        uint64 // 0x30: Target VCN (for nonresident updates)
	TargetAttribute  uint16 // 0x38: Attribute type code
	LCNSegmentOffset uint16 // 0x3A: Offset to LCN segment (if applicable)
	TargetFileRef    uint64 // 0x3C: File reference number (MFT entry)
	TargetAttrID     uint16 // 0x44: Attribute ID
	Alignment        uint16 // 0x46: Padding/alignment
	// Followed by variable-length redo/undo data
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
