package MFT

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	MFTAttributes "github.com/aarsakian/FileSystemForensics/FS/NTFS/MFT/attributes"
	"github.com/aarsakian/FileSystemForensics/img"
	"github.com/aarsakian/FileSystemForensics/logger"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/aarsakian/FileSystemForensics/utils"
)

var RecordSize = 1024

var SIFlags = map[uint32]string{
	1: "Read Only", 2: "Hidden", 4: "System", 32: "Archive", 64: "Device", 128: "Normal",
	256: "Temporary", 512: "Sparse", 1024: "Reparse Point", 2048: "Compressed",
	4096: "Offline",
	8192: "Not Indexed", 16384: "Encrypted",
}

var MFTflags = map[uint16]string{
	0: "File Unallocated", 1: "File Allocated", 2: "Folder Unallocated", 3: "Folder Allocated",
}

type Records []Record

type FixUp struct {
	Signature      []byte
	OriginalValues [][]byte
}

type Attribute interface {
	FindType() string
	SetHeader(header *MFTAttributes.AttributeHeader)
	GetHeader() MFTAttributes.AttributeHeader
	IsNoNResident() bool
	ShowInfo()
	Parse([]byte)
}

// when attributes span over a record entry
type LinkedRecordInfo struct {
	RefEntry uint32
	RefSeq   uint16
	StartVCN uint64
}

// MFT Record
type Record struct {
	Signature            [4]byte //0-3
	UpdateFixUpArrOffset uint16  //4-5      offset values are relative to the start of the entry.
	UpdateFixUpArrSize   uint16  //6-7
	Lsn                  uint64  //8-15    logfile sequence number, points to the most recent LogFile entry for this MFT entry
	Seq                  uint16  //16-17   is incremented when the entry is either allocated or unallocated, determined by the OS.
	Linkcount            uint16  //18-19        how many directories have entries for this MFTentry
	AttrOff              uint16  //20-21       //first attr location
	Flags                uint16  //22-23  //tells whether entry is used or not
	Size                 uint32  //24-27
	AllocSize            uint32  //28-31
	BaseRef              uint64  //32-39
	NextAttrID           uint16  //40-41 e.g. if it is 6 then there are attributes with 1 to 5
	F1                   uint16  //42-43
	Entry                uint32  //44-48                  ??
	FixUp                *FixUp
	Attributes           []Attribute
	LinkedRecordsInfo    []LinkedRecordInfo //holds attrs list entries
	LinkedRecords        []*Record          // when attribute is too long to fit in one MFT record
	OriginLinkedRecord   *Record            // points to the original record that contaisn the attr list
	I30Size              uint64
	Parent               *Record
	// fixupArray add the        UpdateSeqArrOffset to find is location

}
type IndexAttributes interface {
	GetIndexEntriesSortedByMFTEntry() MFTAttributes.IndexEntries
	GetEntries() MFTAttributes.IndexEntries
}

func (mfttable *MFTTable) DetermineClusterOffsetLength() {
	firstRecord := mfttable.Records[0]

	mfttable.Size = int(firstRecord.FindAttribute("DATA").GetHeader().ATRrecordNoNResident.RunListTotalLenCl)

}

func (record Record) IsFolder() bool {
	recordType := record.getType()
	return recordType == "Folder Unallocated" || recordType == "Folder Allocated"
}

func (record *Record) ProcessNoNResidentAttributes(hD img.DiskReader, partitionOffsetB int64, clusterSizeB int) {
	var buf bytes.Buffer
	//allocate a large enough buffer
	buf.Grow(clusterSizeB * 100)
	for idx := range record.Attributes {
		//all non resident attrs except DATA
		//process bitmap
		if !record.Attributes[idx].IsNoNResident() || record.Attributes[idx].FindType() == "DATA" && record.Entry != 6 {
			continue
		}

		attrHeader := record.Attributes[idx].GetHeader()
		if attrHeader.ATRrecordNoNResident == nil {
			continue
		}

		length := int(attrHeader.ATRrecordNoNResident.RunListTotalLenCl) * clusterSizeB
		if length == 0 { // no runlists found
			msg := "non resident attribute has zero length runlist"
			logger.MFTExtractorlogger.Warning(msg)
			continue
		}
		err := attrHeader.ATRrecordNoNResident.GetContent(hD, partitionOffsetB, clusterSizeB, &buf)

		if err != nil {
			continue
		}

		actualLen := int(attrHeader.ATRrecordNoNResident.ActualLength)
		if actualLen > length {
			msg := fmt.Sprintf("attribute  actual length exceeds the runlist length actual %d runlist %d.",
				actualLen, length)
			logger.MFTExtractorlogger.Warning(msg)

		} else {
			record.Attributes[idx].Parse(buf.Bytes()[:actualLen])
		}

		buf.Reset()

	}

}

func (record Record) GetUnallocatedClusters() []int {
	var unallocatedClusters []int
	pos := 0
	bitmap := record.FindAttribute("DATA").(*MFTAttributes.DATA).Content
	for _, byteval := range bitmap {
		bitmask := uint8(0x01)
		shifter := 0
		for bitmask < 128 {

			bitmask = 1 << shifter
			if byteval&bitmask == 0x00 {
				unallocatedClusters = append(unallocatedClusters, pos)
			}
			pos++
			shifter++
		}

	}
	return unallocatedClusters
}

/*func (bitmap BitMap) ShowInfo() {
	fmt.Printf("type %s \n", bitmap.FindType())
	pos := 1
	for _, byteval := range bitmap.AllocationStatus {
		bitmask := uint8(0x01)
		shifter := 0
		for bitmask < 128 {

			bitmask = 1 << shifter
			fmt.Printf("cluster/entry  %d status %d \t", pos, byteval&bitmask)
			pos++
			shifter++
		}

	}
}*/

func (record Record) LocateDataAsync(hD img.DiskReader, partitionOffset int64, clusterSizeB int, dataFragments chan<- []byte) {
	writeOffset := 0
	p := message.NewPrinter(language.Greek)
	if record.HasResidentDataAttr() {
		dataFragments <- record.GetResidentData()

	} else {

		runlist := record.GetRunList("DATA")

		offset := partitionOffset // partition in bytes

		diskSize := hD.GetDiskSize()

		for runlist != nil {

			offset += runlist.Offset * int64(clusterSizeB)
			if offset > diskSize {
				msg := fmt.Sprintf("skipped offset %d exceeds disk size! exiting", offset)
				logger.MFTExtractorlogger.Warning(msg)
				break
			}
			res := p.Sprintf("%d", (offset-partitionOffset)/int64(clusterSizeB))

			msg := fmt.Sprintf("offset %s cl len %d cl.", res, runlist.Length)
			logger.MFTExtractorlogger.Info(msg)
			if runlist.Offset != 0 && runlist.Length > 0 {
				dataFragments <- hD.ReadFile(offset, int(runlist.Length)*clusterSizeB)
			}

			if runlist.Next == nil {
				break
			}

			runlist = runlist.Next
			writeOffset += int(runlist.Length) * clusterSizeB
		}

	}

}

func (record Record) LocateData(hD img.DiskReader, partitionOffset int64, sectorsPerCluster int, bytesPerSector int, results chan<- utils.AskedFile) {
	p := message.NewPrinter(language.Greek)

	writeOffset := 0

	var buf bytes.Buffer

	lSize := int(record.GetLogicalFileSize())
	buf.Grow(lSize)

	if record.HasResidentDataAttr() {
		buf.Write(record.GetResidentData())

	} else {

		runlist := record.GetRunList("DATA")

		offset := partitionOffset // partition in bytes

		diskSize := hD.GetDiskSize()

		for runlist != nil {
			offset += runlist.Offset * int64(sectorsPerCluster*bytesPerSector)
			if offset > diskSize {
				msg := fmt.Sprintf("skipped offset %d exceeds disk size! exiting", offset)
				logger.MFTExtractorlogger.Warning(msg)
				break
			}

			if runlist.Offset != 0 && runlist.Length > 0 {
				buf.Write(hD.ReadFile(offset, int(runlist.Length)*sectorsPerCluster*bytesPerSector))
				res := p.Sprintf("%d", (offset-partitionOffset)/int64(sectorsPerCluster*bytesPerSector))

				msg := fmt.Sprintf("offset %s cl len %d cl.", res, runlist.Length)
				logger.MFTExtractorlogger.Info(msg)
			}

			if runlist.Next == nil {
				break
			}

			runlist = runlist.Next
			writeOffset += int(runlist.Length) * sectorsPerCluster * bytesPerSector
		}

	}
	//truncate buf grows over len?
	results <- utils.AskedFile{Fname: record.GetFname(), Content: buf.Bytes()[:lSize], Id: int(record.Entry)}
}

func (record Record) FindNonResidentAttributes() []Attribute {
	return utils.Filter(record.Attributes, func(attribute Attribute) bool {
		return attribute.IsNoNResident()
	})
}

func (record Record) FilterOutNonResidentAttributes(attrName string) []Attribute {
	return utils.Filter(record.Attributes, func(attribute Attribute) bool {
		return attribute.IsNoNResident() && attribute.FindType() != attrName
	})
}

func (record Record) FindAttributePtr(attributeName string) Attribute {
	for idx := range record.Attributes {
		if record.Attributes[idx].FindType() == attributeName {

			return record.Attributes[idx]
		}
	}
	return nil
}

func (record Record) FindAttribute(attributeName string) Attribute {
	for _, attribute := range record.Attributes {
		if attribute.FindType() == attributeName {

			return attribute
		}
	}
	return nil
}

func (record Record) HasResidentDataAttr() bool {
	attribute := record.FindAttribute("DATA")
	return attribute != nil && !attribute.IsNoNResident()
}

func (record Record) HasNonResidentAttr() bool {
	for _, attr := range record.Attributes {
		if !attr.IsNoNResident() {
			return true
		}
	}
	return false
}

func (record Record) GetLinkedRecords() []*Record {
	return record.LinkedRecords
}

func (record Record) getType() string {
	return MFTflags[record.Flags]
}

func (record Record) GetID() int {
	return int(record.Entry)
}

func (record Record) GetSequence() int {
	return int(record.Seq)
}

func (record Record) IsDeleted() bool {
	return record.getType() == "File Unallocated" || record.getType() == "Folder Unallocated"
}

func (record Record) GetRunList(attrType string) *MFTAttributes.RunList {
	if len(record.LinkedRecords) == 0 {
		attr := record.FindAttribute(attrType)
		return attr.GetHeader().ATRrecordNoNResident.RunList
	} else {
		attr := record.LinkedRecords[0].FindAttribute(attrType)
		return attr.GetHeader().ATRrecordNoNResident.RunList
	}

}

func (record Record) GetRunLists() []MFTAttributes.RunList {
	var runlists []MFTAttributes.RunList
	for _, attribute := range record.Attributes {
		if attribute.IsNoNResident() {

			runlists = append(runlists, *attribute.GetHeader().ATRrecordNoNResident.RunList)
		}
	}

	return runlists
}

func (record Record) GetFullPath() string {
	var fullpath strings.Builder
	parent := record.Parent
	for parent != nil && parent.Entry != 5 { //$MFT Root entry
		//prepends
		fullpath.WriteRune(os.PathSeparator)
		fullpath.WriteString(parent.GetFname())
		parent = parent.Parent
	}
	//reverse

	return filepath.Join(fullpath.String())
}

func (record Record) ShowVCNs() {
	startVCN, lastVCN := record.getVCNs()
	if startVCN != 0 || lastVCN != 0 {
		fmt.Printf(" startVCN %d endVCN %d ", startVCN, lastVCN)
	}

}

func (record Record) ShowParentRecordInfo() {
	if record.Parent == nil {
		fmt.Printf("\n Record has no parent ")
	} else {
		fmt.Printf("\n Record has parent ")
		record.Parent.showInfo()
		record.Parent.ShowFileName("win32")

	}

}

func (record Record) ShowPath(partitionId int) {
	fullpath := record.GetFullPath()
	fmt.Printf("Partition%d%s\\%s\n", partitionId, fullpath, record.GetFname())
}

func (record Record) ShowIndex() {
	indexAttr := record.FindAttribute("Index Root")
	indexAlloc := record.FindAttribute("Index Allocation")

	if indexAttr != nil {
		idxRoot := indexAttr.(*MFTAttributes.IndexRoot)
		idxRoot.ShowInfo()

	}

	if indexAlloc != nil {
		idx := indexAlloc.(*MFTAttributes.IndexAllocation)
		idx.ShowInfo()

	}

}

func (record Record) getVCNs() (uint64, uint64) {
	for _, attribute := range record.Attributes {
		if attribute.IsNoNResident() {
			return attribute.GetHeader().ATRrecordNoNResident.StartVcn,
				attribute.GetHeader().ATRrecordNoNResident.LastVcn
		}
	}
	return 0, 0

}

func (record Record) ShowAttributes(attrType string) {
	fmt.Printf("ID %d  ", record.Entry)
	var attributes []Attribute
	if attrType == "any" {
		attributes = record.Attributes
	} else {
		attributes = utils.Filter(record.Attributes, func(attribute Attribute) bool {
			return attribute.FindType() == attrType
		})
	}

	for _, attribute := range attributes {
		attribute.ShowInfo()
	}

}

func (record Record) ShowTimestamps() {
	var attr Attribute
	attr = record.FindAttribute("FileName")
	if attr != nil {
		fnattr := attr.(*MFTAttributes.FNAttribute)
		atime, ctime, mtime, mftime := fnattr.GetTimestamps()
		fmt.Printf("FN a %s c %s m %s mftm %s ", atime, ctime, mtime, mftime)
	}
	attr = record.FindAttribute("Standard Information")
	if attr != nil {
		siattr := attr.(*MFTAttributes.SIAttribute)
		atime, ctime, mtime, mftime := siattr.GetTimestamps()
		fmt.Printf("SI a %s c %s m %s mftm %s ", atime, ctime, mtime, mftime)
	}
	//get parent
	if !record.IsFolder() && record.Parent != nil {

		record.Parent.ShowIndexTimestamps("Index Root")
		record.Parent.ShowIndexTimestamps("Index Allocation")

	}

}

func (record Record) ShowIndexTimestamps(attrName string) {
	attr := record.FindAttribute(attrName)

	if attr != nil {

		for _, entry := range attr.(IndexAttributes).GetEntries() {
			if entry.Fnattr == nil || entry.ParRef != uint64(record.Entry) {
				continue
			}

			atime, ctime, mtime, mftime := entry.Fnattr.GetTimestamps()
			fmt.Printf("%s a %s c %s m %s mftm %s ", attrName, atime, ctime, mtime, mftime)
		}

	}
}

func (record Record) showInfo() {
	fmt.Printf("record %d type %s\n", record.Entry, record.getType())
}

func (record Record) GetResidentData() []byte {
	return record.FindAttribute("DATA").(*MFTAttributes.DATA).Content

}

func (record Record) ShowRunList() {
	runlists := record.GetRunLists()
	nonResidentAttributes := record.FindNonResidentAttributes()

	for _, linkedRecord := range record.LinkedRecords {
		runlists = append(runlists, linkedRecord.GetRunLists()...)
		nonResidentAttributes = append(nonResidentAttributes, linkedRecord.FindNonResidentAttributes()...)
	}

	for idx, nonResidentAttr := range nonResidentAttributes {
		fmt.Printf("%s \n", nonResidentAttr.FindType())
		runlist := runlists[idx]
		logicalOffset := runlist.Offset
		nofFragments := 0
		totalClusters := 0
		for (MFTAttributes.RunList{}) != runlist {

			fmt.Printf(" offs. %d cl len %d cl  logical offset %d cl clusters %d \n",
				runlist.Offset, runlist.Length, logicalOffset, totalClusters)
			if runlist.Next == nil {
				break
			}
			runlist = *runlist.Next
			logicalOffset += runlist.Offset
			totalClusters += int(runlist.Length)
			nofFragments += 1
		}
		fmt.Printf("Total Clusters %d Total Fragments %d\n", totalClusters, nofFragments)

	}

}

func (record Record) HasFilenameExtension(extension string) bool {
	if record.HasAttr("FileName") {
		fnattr := record.FindAttribute("FileName").(*MFTAttributes.FNAttribute)
		if strings.HasSuffix(fnattr.Fname, strings.ToUpper("."+extension)) ||
			strings.HasSuffix(fnattr.Fname, strings.ToLower("."+extension)) {
			return true
		}
	}

	return false
}

func (record Record) HasFilename(filename string) bool {
	return record.GetFname() == filename

}

func (record Record) HasFilenames(filenames []string) bool {
	for _, filename := range filenames {
		if record.HasFilename(filename) {
			return true
		}
	}
	return false

}

func (record Record) HasPath(filespath string) bool {
	return record.GetFullPath() == filespath

}

func (record Record) HasAttr(attrName string) bool {
	return record.FindAttribute(attrName) != nil
}

func (record Record) ShowIsResident() {
	if record.HasAttr("DATA") {
		if record.HasResidentDataAttr() {
			fmt.Printf("Resident")
		} else {
			fmt.Printf("NoN Resident")
		}

	} else {
		fmt.Print("NO DATA attr")
	}
}

func (record Record) ShowFNAModifiedTime() {
	fnattr := record.FindAttribute("FileName").(*MFTAttributes.FNAttribute)
	fmt.Printf("%s ", fnattr.Mtime.ConvertToIsoTime())
}

func (record Record) ShowFNACreationTime() {
	fnattr := record.FindAttribute("FileName").(*MFTAttributes.FNAttribute)
	fmt.Printf("%s ", fnattr.Crtime.ConvertToIsoTime())
}

func (record Record) ShowFNAMFTModifiedTime() {
	fnattr := record.FindAttribute("FileName").(*MFTAttributes.FNAttribute)
	fmt.Printf("%s ", fnattr.MFTmtime.ConvertToIsoTime())
}

func (record Record) ShowFNAMFTAccessTime() {
	fnattr := record.FindAttribute("FileName").(*MFTAttributes.FNAttribute)
	fmt.Printf("%s ", fnattr.Atime.ConvertToIsoTime())
}

func (record Record) GetSignature() string {
	return string(record.Signature[:])
}

func (record *Record) ProcessFixUpArrays(data []byte) {
	if len(data) < int(2*record.UpdateFixUpArrSize) {
		msg := fmt.Sprintf("Data not enough to parse fixup array by %d", int(2*record.UpdateFixUpArrSize)-len(data))
		logger.MFTExtractorlogger.Warning(msg)
		return
	}
	fixuparray := data[record.UpdateFixUpArrOffset : record.UpdateFixUpArrOffset+2*record.UpdateFixUpArrSize]
	var fixupvals [][]byte
	val := 2
	for val < len(fixuparray) {

		fixupvals = append(fixupvals, fixuparray[val:val+2])
		val += 2
	}
	record.FixUp = &FixUp{Signature: fixuparray[:2], OriginalValues: fixupvals}

}

func (record *Record) Process(bs []byte) error {

	utils.Unmarshal(bs, record)

	if record.GetSignature() == "BAAD" { //skip bad entry
		return errors.New("Record is corrupt")
	} else if record.GetSignature() != "FILE" {
		return fmt.Errorf("Record has non valid signature %x", record.GetSignature())
	}
	record.ProcessFixUpArrays(bs)
	record.I30Size = 0 //default value

	ReadPtr := record.AttrOff //offset to first attribute
	var linkedRecordsInfo []LinkedRecordInfo
	var attributes []Attribute

	//fixup check
	if record.FixUp.Signature[0] == bs[510] && record.FixUp.Signature[1] == bs[511] {
		bs[510] = record.FixUp.OriginalValues[0][0]
		bs[511] = record.FixUp.OriginalValues[0][1]
	}

	for ReadPtr < 1024 {

		if utils.Hexify(bs[ReadPtr:ReadPtr+4]) == "ffffffff" { //End of attributes
			break
		}

		var attrHeader MFTAttributes.AttributeHeader
		utils.Unmarshal(bs[ReadPtr:ReadPtr+16], &attrHeader)

		if attrHeader.IsLast() { // End of attributes
			break
		}

		if !attrHeader.IsNoNResident() { //Resident Attribute
			var attr Attribute

			var atrRecordResident *MFTAttributes.ATRrecordResident = new(MFTAttributes.ATRrecordResident)

			atrRecordResident.Parse(bs[ReadPtr+16:])
			atrRecordResident.Name = utils.DecodeUTF16(bs[ReadPtr+attrHeader.NameOff : ReadPtr+attrHeader.NameOff+2*uint16(attrHeader.Nlen)])
			attrHeader.ATRrecordResident = atrRecordResident
			attrStartOffset := ReadPtr + atrRecordResident.OffsetContent
			attrEndOffset := uint32(attrStartOffset) + atrRecordResident.ContentSize

			if attrHeader.IsFileName() { // File name
				attr = &MFTAttributes.FNAttribute{}
				attr.Parse(bs[attrStartOffset:attrEndOffset])

			} else if attrHeader.IsReparse() {
				attr = &MFTAttributes.Reparse{}
				attr.Parse(bs[attrStartOffset:attrEndOffset])

			} else if attrHeader.IsData() {
				attr = &MFTAttributes.DATA{}
				attr.Parse(bs[attrStartOffset:attrEndOffset])

			} else if attrHeader.IsObject() {
				attr = &MFTAttributes.ObjectID{}
				attr.Parse(bs[attrStartOffset:attrEndOffset])
			} else if attrHeader.IsAttrList() { //Attribute List

				attr = &MFTAttributes.AttributeListEntries{}

				attr.Parse(bs[attrStartOffset:attrEndOffset])
				attrListEntries := attr.(*MFTAttributes.AttributeListEntries) //dereference

				for _, entry := range attrListEntries.Entries {
					if entry.GetType() != "DATA" {
						continue
					}
					linkedRecordsInfo = append(linkedRecordsInfo,
						LinkedRecordInfo{RefEntry: uint32(entry.ParRef), StartVCN: entry.StartVcn, RefSeq: entry.ParSeq})
				}

			} else if attrHeader.IsBitmap() { //BITMAP
				attr = &MFTAttributes.BitMap{}
				attr.Parse(bs[attrStartOffset:attrEndOffset])

			} else if attrHeader.IsVolumeName() { //Volume Name
				attr = &MFTAttributes.VolumeName{}
				attr.Parse(bs[attrStartOffset:attrEndOffset])

			} else if attrHeader.IsVolumeInfo() { //Volume Info
				attr = &MFTAttributes.VolumeInfo{}
				attr.Parse(bs[attrStartOffset:attrEndOffset])

			} else if attrHeader.IsIndexRoot() { //Index Root
				attr = &MFTAttributes.IndexRoot{}
				attr.Parse(bs[attrStartOffset:attrEndOffset])

			} else if attrHeader.IsStdInfo() { //Standard Information

				attr = &MFTAttributes.SIAttribute{}
				attr.Parse(bs[attrStartOffset:attrEndOffset])

			} else if attrHeader.IsLoggedUtility() {
				attr = &MFTAttributes.LoggedUtilityStream{Kind: attrHeader.GetName()}
				attr.Parse(bs[attrStartOffset:attrEndOffset])

			} else if attrHeader.IsExtendedAttribute() {
				attr = &MFTAttributes.ExtendedAttribute{}
				attr.Parse(bs[attrStartOffset:attrEndOffset])
			} else if attrHeader.IsExtendedInformationAttribute() {
				attr = &MFTAttributes.EA_INFORMATION{}
				attr.Parse(bs[attrStartOffset:attrEndOffset])
			} else {
				msg := fmt.Sprintf("uknown resident attribute %s at record %d",
					attrHeader.GetType(), record.Entry)
				logger.MFTExtractorlogger.Warning(msg)
			}
			if attr != nil {
				attr.SetHeader(&attrHeader)
				attributes = append(attributes, attr)

			}

		} else { //NoN Resident Attribute
			var atrNoNRecordResident *MFTAttributes.ATRrecordNoNResident = new(MFTAttributes.ATRrecordNoNResident)
			utils.Unmarshal(bs[ReadPtr+16:ReadPtr+64], atrNoNRecordResident)

			if int(ReadPtr+atrNoNRecordResident.RunOff+attrHeader.AttrLen) < len(bs) {
				var runlist *MFTAttributes.RunList = new(MFTAttributes.RunList)
				lengthcl := runlist.Process(bs[ReadPtr+
					atrNoNRecordResident.RunOff : ReadPtr+attrHeader.AttrLen])
				atrNoNRecordResident.RunList = runlist
				atrNoNRecordResident.RunListTotalLenCl = lengthcl

			} else {
				msg := fmt.Sprintf("attribute %s at record %d exceeded buffer by %d",
					attrHeader.GetType(), record.Entry, int(ReadPtr+attrHeader.AttrLen)-len(bs))
				logger.MFTExtractorlogger.Warning(msg)
			}
			attrHeader.ATRrecordNoNResident = atrNoNRecordResident

			if attrHeader.IsData() {
				data := &MFTAttributes.DATA{}
				data.SetHeader(&attrHeader)
				attributes = append(attributes, data)
			} else if attrHeader.IsIndexAllocation() {
				var idxAllocation *MFTAttributes.IndexAllocation = new(MFTAttributes.IndexAllocation)
				idxAllocation.SetHeader(&attrHeader)
				attributes = append(attributes, idxAllocation)
			} else if attrHeader.IsBitmap() { //BITMAP
				var bitmap *MFTAttributes.BitMap = new(MFTAttributes.BitMap)
				bitmap.SetHeader(&attrHeader)
				attributes = append(attributes, bitmap)
			} else if attrHeader.IsAttrList() {
				var attrListEntries *MFTAttributes.AttributeListEntries = new(MFTAttributes.AttributeListEntries)
				attrListEntries.SetHeader(&attrHeader)
				attributes = append(attributes, attrListEntries)
			} else if attrHeader.IsReparse() {
				var reparse *MFTAttributes.Reparse = new(MFTAttributes.Reparse)
				reparse.SetHeader(&attrHeader)
				attributes = append(attributes, reparse)
			} else {
				msg := fmt.Sprintf("unknown non resident attr %s", attrHeader.GetType())
				logger.MFTExtractorlogger.Warning(msg)
			}

		} //ends non Resident
		ReadPtr = ReadPtr + uint16(attrHeader.AttrLen)

	} //ends while
	record.Attributes = attributes
	record.LinkedRecordsInfo = linkedRecordsInfo
	return nil
}

func (record Record) ShowFileSize() {

	logical := record.GetLogicalFileSize()
	physical := record.GetPhysicalSize()
	fmt.Printf(" logical: %d (KB), physical: %d (KB)",
		logical/1024, physical/1024)

}

func (record Record) GetPhysicalSize() int64 {
	attr := record.FindAttribute("FileName")
	if attr != nil {
		return int64(attr.(*MFTAttributes.FNAttribute).AllocFsize)
	} else {
		return 0
	}

}

func (record Record) GetLogicalFileSize() int64 {
	if record.OriginLinkedRecord != nil {
		return record.OriginLinkedRecord.GetLogicalFileSize()
	}
	attr := record.FindAttribute("FileName")
	if attr != nil {
		fnattr := attr.(*MFTAttributes.FNAttribute)
		if fnattr.RealFsize != 0 {
			return int64(fnattr.RealFsize)
		}

	}
	return int64(record.I30Size)
}

func (record Record) GetFnames() map[string]string {

	fnAttributes := utils.Filter(record.Attributes, func(attribute Attribute) bool {
		return attribute.FindType() == "FileName"
	})
	fnames := make(map[string]string, len(fnAttributes))
	for _, attr := range fnAttributes {
		fnattr := attr.(*MFTAttributes.FNAttribute)
		fnames[fnattr.GetFileNameType()] = fnattr.Fname

	}

	return fnames

}

func (record Record) GetFname() string {
	if record.OriginLinkedRecord != nil {
		return record.OriginLinkedRecord.GetFname()
	}
	fnames := record.GetFnames()
	for _, namescheme := range []string{"Win32", "Win32 & Dos", "POSIX", "Dos"} {
		name, ok := fnames[namescheme]
		if ok {
			return name
		}
	}
	return "-"

}

func (record Record) ShowFileName(fileNameSyntax string) {

	fnames := record.GetFnames()
	for ftype, fname := range fnames {
		if ftype == fileNameSyntax {
			fmt.Printf(" %s ", fname)
		} else {
			fmt.Printf(" %s ", fname)
		}
	}
}

func (record Record) HasParent() bool {
	return record.Parent != nil
}

func (records Records) GetParent(record Record) (Record, error) {
	fnattr := record.FindAttribute("FileName").(*MFTAttributes.FNAttribute)
	if fnattr == nil {
		return Record{}, errors.New("no filename attribute located")
	} else {
		if records[fnattr.ParRef].Seq-fnattr.ParSeq < 2 { //record is children
			return records[fnattr.ParRef], nil
		} else {
			return Record{}, errors.New("parent has been reallocated")
		}

	}

}

func (record Record) HasPrefix(prefix string) bool {
	record_name := record.GetFname()
	return strings.HasPrefix(record_name, prefix)
}

func (record Record) HasSuffix(suffix string) bool {
	record_name := record.GetFname()
	return strings.HasSuffix(record_name, suffix)
}
