package UsnJrnl

/*USN record describes, on a higher level than
the Logfile does, what happened to a file (or directory, all the same)*/

import (
	"errors"
	"fmt"
	"sync"

	"github.com/aarsakian/MFTExtractor/FS/NTFS/MFT"
	"github.com/aarsakian/MFTExtractor/disk"
	"github.com/aarsakian/MFTExtractor/logger"
	"github.com/aarsakian/MFTExtractor/utils"
)

var reasons = map[uint32]string{
	0x00008000: "USN_REASON_BASIC_INFO_CHANGE",
	0x80000000: "USN_REASON_CLOSE",
	0x00020000: "USN_REASON_COMPRESSION_CHANGE",
	0x00000002: "USN_REASON_DATA_EXTEND",
	0x00000001: "USN_REASON_DATA_OVERWRITE",
	0x00000004: "USN_REASON_DATA_TRUNCATION",
	0x00000400: "USN_REASON_EA_CHANGE",
	0x00040000: "USN_REASON_ENCRYPTION_CHANGE",
	0x00000100: "USN_REASON_FILE_CREATE",
	0x00000200: "USN_REASON_FILE_DELETE",
	0x00010000: "USN_REASON_HARD_LINK_CHANGE",
	0x00004000: "USN_REASON_INDEXABLE_CHANGE",
	0x00800000: "USN_REASON_INTEGRITY_CHANGE",
	0x00000020: "USN_REASON_NAMED_DATA_EXTEND",
	0x00000010: "USN_REASON_NAMED_DATA_OVERWRITE",
	0x00000040: "USN_REASON_NAMED_DATA_TRUNCATION",
	0x00080000: "USN_REASON_OBJECT_ID_CHANGE",
	0x00002000: "USN_REASON_RENAME_NEW_NAME",
	0x00001000: "USN_REASON_RENAME_OLD_NAME",
	0x00100000: "USN_REASON_REPARSE_POINT_CHANGE",
	0x00000800: "USN_REASON_SECURITY_CHANGE",
	0x00200000: "USN_REASON_STREAM_CHANGE",
	0x00400000: "USN_REASON_TRANSACTED_CHANGE"}

var source = map[uint32]string{
	0x00000001: "USN_SOURCE_DATA_MANAGEMENT",
	0x00000002: "USN_SOURCE_AUXILIARY_DATA",
	0x00000004: "USN_SOURCE_REPLICATION_MANAGEMENT",
	0x00000008: "USN_SOURCE_CLIENT_REPLICATION_MANAGEMENT",
}

var fileattributes = map[uint32]string{
	1: "Read Only", 2: "Hidden", 4: "System",
	32: "Archive", 64: "Device", 128: "Normal", 256: "Temporary", 512: "Sparse file",
	1024: "Reparse", 2048: "Compressed", 4096: "Offline",
	8192:    "Content  is not being indexed for faster searches",
	16384:   "Encrypted",
	32768:   "FILE_ATTRIBUTE_INTEGRITY_STREAM",
	65536:   "FILE_ATTRIBUTE_VIRTUAL",
	131072:  "FILE_ATTRIBUTE_NO_SCRUB_DATA",
	262144:  "FILE_ATTRIBUTE_EA",
	524288:  "FILE_ATTRIBUTE_PINNED",
	1048576: "FILE_ATTRIBUTE_UNPINNED",
	2097152: "FILE_ATTRIBUTE_RECALL_ON_OPEN",
	4194304: "FILE_ATTRIBUTE_RECALL_ON_DATA_ACCESS",
}

type Records []Record

type Record struct {
	Length      uint32
	MajorVer    uint16 //2-> USN v2, 3-> USN v3
	MinorVer    uint16
	EntryRef    uint64
	EntrySeq    uint16
	ParRef      uint64
	ParSeq      uint16
	USN         uint64
	EventTime   utils.WindowsTime
	ReasonFlag  uint32
	SourceInfo  uint32
	SecurityId  uint32
	FileAttrs   uint32
	FnameLen    uint16 //length of name
	FnameOffset uint16 //format of name 58-60
	Fname       string //special string type without nulls

}

func Process(mftrecords MFT.Records, disk disk.Disk, partitionId int) Records {

	var records Records
	for _, record := range mftrecords {
		recordsCH := make(chan Record)
		wg := new(sync.WaitGroup)
		wg.Add(2)
		dataClusters := make(chan []byte, record.GetLogicalFileSize())

		go disk.AsyncWorker(wg, record, dataClusters, partitionId)
		go AsyncProcess(wg, dataClusters, recordsCH)
		for record := range recordsCH {
			records = append(records, record)
		}

		wg.Wait()
	}

	return records
}

func AsyncProcess(wg *sync.WaitGroup, dataClusters <-chan []byte, recordsCH chan<- Record) {
	defer wg.Done()

	for dataCluster := range dataClusters {
		offset := 0
		for offset < len(dataCluster) {

			msg := fmt.Sprintf("USN record at rel %d", offset)
			logger.MFTExtractorlogger.Info(msg)
			record := new(Record)
			parsedLen, err := record.Parse(dataCluster[offset:])

			if err != nil {
				msg := fmt.Sprintf("Parsing usnjrnl exceed available buffer by %d at offset %d",
					parsedLen-len(dataCluster[offset:]), offset)
				logger.MFTExtractorlogger.Warning(msg)
				break
			}
			offset += parsedLen

			if record.EntryRef == 0 {
				continue
			}

			recordsCH <- *record
		}
	}
	close(recordsCH)

}

func (record *Record) Parse(data []byte) (int, error) {
	readFrom := 60
	if len(data) < readFrom {
		return readFrom, errors.New("not enough data to unmarshal record usnjrnl")
	}

	utils.Unmarshal(data, record)

	readTo := 60 + 2*int(record.FnameLen)
	if readTo > len(data) {
		return readTo, errors.New("exceeded available data")
	}
	record.Fname = utils.DecodeUTF16(data[60:readTo])
	return readTo, nil
}

func (record Record) GetSourceInfo() string {
	return source[record.SourceInfo]
}

func (record Record) GetFileAttributes() string {
	return fileattributes[record.FileAttrs]
}

func (record Record) GetReason() string {
	return reasons[record.ReasonFlag]
}

func (record Record) ShowInfo() {
	fmt.Printf("%s %s %s entry Ref %d entry Seq %d parent Ref %d parent Seq %d\n",
		record.Fname, record.GetReason(), record.GetFileAttributes(), record.EntryRef, record.EntrySeq,
		record.ParRef, record.ParSeq)
}
