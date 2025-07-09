package MFT

import (
	"bytes"
	"errors"
	"fmt"
	"sync"

	MFTAttributes "github.com/aarsakian/FileSystemForensics/FS/NTFS/MFT/attributes"
	"github.com/aarsakian/FileSystemForensics/img"
	"github.com/aarsakian/FileSystemForensics/logger"
	"github.com/aarsakian/FileSystemForensics/utils"
)

type ParentReallocatedError struct {
	input string
}

func (e *ParentReallocatedError) Error() string {
	return e.input
}

// $MFT table points either to its file path or the buffer containing $MFT
type MFTTable struct {
	Records []Record
	Size    int
}

func (mfttable *MFTTable) ProcessRecordsAsync(data []byte) {

	var wg sync.WaitGroup
	mfttable.Records = make([]Record, len(data)/RecordSize)
	msg := fmt.Sprintf("Processing %d $MFT entries", len(mfttable.Records))
	fmt.Printf(" %s \n", msg)
	logger.FSLogger.Info(msg)

	for i := 0; i < len(data); i += RecordSize {
		if utils.Hexify(data[i:i+4]) == "00000000" { //zero area skip
			continue
		}
		wg.Add(1)
		go mfttable.MFTWorker(data[i:i+RecordSize], i, &wg)

	}
	wg.Wait()
}

func (mfttable *MFTTable) ProcessRecords(data []byte) {

	mfttable.Records = make([]Record, len(data)/RecordSize)
	msg := fmt.Sprintf("Processing %d $MFT entries", len(mfttable.Records))
	fmt.Printf(" %s \n", msg)
	logger.FSLogger.Info(msg)

	var record Record
	for i := 0; i < len(data); i += RecordSize {
		if utils.Hexify(data[i:i+4]) == "00000000" { //zero area skip
			continue
		}

		err := record.Process(data[i : i+RecordSize])
		if err != nil {
			logger.FSLogger.Error(err)
			continue
		}

		mfttable.Records[record.Entry] = record

		logger.FSLogger.Info(fmt.Sprintf("Processed record %d at pos %d", record.Entry, i/RecordSize))

	}

}

func (mfttable *MFTTable) MFTWorker(data []byte, pos int, wg *sync.WaitGroup) {
	defer wg.Done()
	var record Record
	err := record.Process(data)
	if err != nil {
		logger.FSLogger.Error(err)
		return
	}

	mfttable.Records[record.Entry] = record

	logger.FSLogger.Info(fmt.Sprintf("Processed record %d at pos %d", record.Entry, pos/RecordSize))
}

func (mfttable *MFTTable) ProcessNonResidentRecords(hD img.DiskReader, partitionOffsetB int64, clusterSizeB int) {
	//2 * runtime.NumCPU()
	numWorker := 4

	records := make(chan Record, len(mfttable.Records))

	var wg sync.WaitGroup
	for w := 1; w <= numWorker; w++ {
		wg.Add(1)
		go ProcessNoNResidentAttributesWorker(records, hD, partitionOffsetB, clusterSizeB, &wg)
	}
	for idx := range mfttable.Records {
		records <- mfttable.Records[idx]
	}

	close(records)
	wg.Wait()

}

func (mfttable *MFTTable) ProcessNonResidentRecordsSync(hD img.DiskReader, partitionOffsetB int64, clusterSizeB int) int {
	totalReadBytes := 0
	var buf bytes.Buffer
	//allocate a large enough buffer
	buf.Grow(clusterSizeB)

	for idx := range mfttable.Records {
		totalReadBytes += mfttable.Records[idx].ProcessNoNResidentAttributes(hD, partitionOffsetB, clusterSizeB, &buf)
	}
	return totalReadBytes
}

func (mfttable *MFTTable) CreateLinkedRecords() {
	//recreate chain  for fragmented $MFT records (attrList present)
	for idx := range mfttable.Records {

		for _, linkedRecordInfo := range mfttable.Records[idx].LinkedRecordsInfo {
			//cannot point to itself
			if mfttable.Records[idx].Entry == linkedRecordInfo.RefEntry {
				continue
			}

			linkedRecord, err := mfttable.GetRecord(linkedRecordInfo.RefEntry, linkedRecordInfo.RefSeq)

			if err != nil {
				continue
			}
			logger.FSLogger.Info(fmt.Sprintf("updated linked record %d", linkedRecord.Entry))

			linkedRecord.OriginLinkedRecord = &mfttable.Records[idx]
			mfttable.Records[idx].LinkedRecords = append(mfttable.Records[idx].LinkedRecords, linkedRecord)

		}
	}
}

func (mfttable *MFTTable) FindParentRecords() {

	for idx := range mfttable.Records {
		attr := mfttable.Records[idx].FindAttribute("FileName")
		if attr == nil {
			//logger.FSLogger.Warning(fmt.Sprintf("No FileName attribute found at record %d ", mfttable.Records[idx].Entry))
			continue

		}
		fnattr := attr.(*MFTAttributes.FNAttribute)
		parentRecord, err := mfttable.GetRecord(uint32(fnattr.ParRef), fnattr.ParSeq)

		if err != nil {
			continue
		}

		logger.FSLogger.Info(fmt.Sprintf("update record %d with parent %d", mfttable.Records[idx].Entry, parentRecord.Entry))
		mfttable.Records[idx].Parent = parentRecord

	}
}

func (mfttable MFTTable) GetRecord(referencedEntry uint32, referencedSeq uint16) (*Record, error) {
	if int(referencedEntry) < len(mfttable.Records) {
		if mfttable.Records[referencedEntry].Entry == referencedEntry {

			if mfttable.Records[referencedEntry].Seq-referencedSeq > 1 { //allow for deleted records
				msg := fmt.Sprintf("entry %d has been reallocated with seq %d ref seq %d", mfttable.Records[referencedEntry].Entry,
					mfttable.Records[referencedEntry].Seq, referencedSeq)
				logger.FSLogger.Warning(msg)
				return nil, fmt.Errorf("%s", ParentReallocatedError{msg})
			} else {
				return &mfttable.Records[referencedEntry], nil
			}
		}
	}
	msg := fmt.Sprintf("cannot find entry record for ref %d", referencedEntry)
	logger.FSLogger.Warning(msg)
	return nil, errors.New(msg)
}

func (mfttable *MFTTable) CalculateFileSizes() {
	numWorker := 4
	recordIds := make(chan int, len(mfttable.Records))

	var wg sync.WaitGroup
	for w := 1; w <= numWorker; w++ {
		wg.Add(1)
		go func(wg *sync.WaitGroup, recordIds chan int) {
			defer wg.Done()
			for idx := range recordIds {
				if !mfttable.Records[idx].IsFolder() {
					continue
				}

				if mfttable.Records[idx].HasAttr("Index Root") {
					mfttable.SetI30Size(idx, "Index Root")
				}
				if mfttable.Records[idx].HasAttr("Index Allocation") {
					mfttable.SetI30Size(idx, "Index Allocation")
				}

			}

		}(&wg, recordIds)
	}
	for _, record := range mfttable.Records {
		recordIds <- int(record.Entry)
	}

	close(recordIds)
	wg.Wait()

}

func (mfttable *MFTTable) SetI30Size(recordId int, attrType string) {

	attr := mfttable.Records[recordId].FindAttribute(attrType).(IndexAttributes)

	idxEntries := attr.GetIndexEntriesSortedByMFTEntry()

	for _, idxEntry := range idxEntries {

		//issue with realsize in 8.3 fnattr
		referencedEntry, err := mfttable.GetRecord(uint32(idxEntry.ParRef), idxEntry.ParSeq)

		if err != nil {
			continue
		}

		// set file size omit folders
		if referencedEntry.IsFolder() {
			continue
		}

		logger.FSLogger.Info(fmt.Sprintf("updated I30 size of ref Entry %d", referencedEntry.Entry))

		if idxEntry.Fnattr.RealFsize > idxEntry.Fnattr.AllocFsize {

			referencedEntry.I30Size = idxEntry.Fnattr.AllocFsize
		} else {
			referencedEntry.I30Size = idxEntry.Fnattr.RealFsize
		}

	}

}
