package fstree

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/aarsakian/FileSystemForensics/FS/BTRFS/attributes"
	"github.com/aarsakian/FileSystemForensics/FS/BTRFS/leafnode"
	"github.com/aarsakian/FileSystemForensics/img"
	"github.com/aarsakian/FileSystemForensics/logger"
	"github.com/aarsakian/FileSystemForensics/utils"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type FilesDirsMap map[uint64]FileDirEntry //inodeid -> FileDirEntry

type FileDirEntry struct {
	Id       int
	Index    int
	Children []*FileDirEntry
	Parent   *FileDirEntry
	Path     string

	DataItems []leafnode.DataItem
	Items     []leafnode.Item
}

func (fileDirEntry FileDirEntry) GetParentId() (int, error) {
	if fileDirEntry.Parent != nil {
		return fileDirEntry.Parent.Id, nil
	} else {
		return -1, errors.New("no parent found")
	}
}

func (fileDirEntry *FileDirEntry) GetExtents() map[uint64]*attributes.ExtentData {
	//	Logical offset in the file to extent
	var extentsMap map[uint64]*attributes.ExtentData
	for idx, item := range fileDirEntry.Items {
		if !item.IsExtentData() {
			continue
		}
		extData, ok := fileDirEntry.DataItems[idx].(*attributes.ExtentData)
		if !ok {
			continue
		}

		extentsMap[item.Key.Offset] = extData
	}
	return extentsMap

}

func (fileDirEntry FileDirEntry) GetLogicalFileSize() int64 {
	attr := fileDirEntry.FindAttributes("INODE_ITEM")[0].(*attributes.InodeItem)
	return int64(attr.StSize)
}

func (fileDirEntry FileDirEntry) GetFname() string {
	attrs := fileDirEntry.FindAttributes("INODE_REF")
	if len(attrs) == 0 {
		logger.FSLogger.Warning("no inode_ref attribute found")
		return ""
	}
	attr := attrs[0].(*attributes.InodeRef)
	return attr.Name
}

func (fileDirEntry FileDirEntry) GetID() int {
	return fileDirEntry.Id
}

func (fileDirEntry FileDirEntry) GetLinkedRecords() []*FileDirEntry {
	return []*FileDirEntry{}
}

func (fileDirEntry FileDirEntry) GetSequence() int {
	return 0
}

func (fileDirEntry FileDirEntry) HasFilenameExtension(extension string) bool {
	fname := fileDirEntry.GetFname()
	if strings.HasSuffix(fname, strings.ToUpper("."+extension)) ||
		strings.HasSuffix(fname, strings.ToLower("."+extension)) {
		return true
	}

	return false
}

func (fileDirEntry FileDirEntry) HasFilenames(filenames []string) bool {
	fname := fileDirEntry.GetFname()
	for _, filename := range filenames {
		if fname == filename {
			return true
		}
	}
	return false
}

func (fileDirEntry FileDirEntry) HasParent() bool {
	return fileDirEntry.Parent != nil
}

func (fileDirEntry FileDirEntry) HasPath(path string) bool {
	return fileDirEntry.Path == path
}

func (fileDirEntry FileDirEntry) HasPrefix(prefix string) bool {
	fname := fileDirEntry.GetFname()
	return strings.HasPrefix(fname, prefix)
}

func (fileDirEntry FileDirEntry) HasSuffix(suffix string) bool {
	fname := fileDirEntry.GetFname()
	return strings.HasSuffix(fname, suffix)
}

func (fileDirEntry FileDirEntry) IsDeleted() bool {
	return false
}

func (fileDirEntry FileDirEntry) IsFolder() bool {
	return false
}

func (fileDirEntry FileDirEntry) GetIndex() int {
	attr := fileDirEntry.FindAttributes("INODE_REF")[0].(*attributes.InodeRef)
	return int(attr.Index)
}

func (fileDirEntry FileDirEntry) LocateDataAsync(hD img.DiskReader, partitionOffset int64,
	clusterSizeB int, dataFragments chan<- []byte) {

}

func (fileDirEntry FileDirEntry) LocateData(hD img.DiskReader, partitionOffset int64, sectorsPerCluster int, bytesPerSector int, results chan<- utils.AskedFile) {
	p := message.NewPrinter(language.Greek)

	var buf bytes.Buffer

	lSize := int(fileDirEntry.GetLogicalFileSize())
	buf.Grow(lSize)

	/*if fileDirEntry.HasResidentDataAttr() {
		buf.Write(fileDirEntry.GetResidentData())

	} else {*/

	diskSize := hD.GetDiskSize()

	for logicalOffset, extent := range fileDirEntry.GetExtents() {
		offset := partitionOffset + int64(logicalOffset)
		if offset > diskSize {
			msg := fmt.Sprintf("skipped offset %d exceeds disk size! exiting", offset)
			logger.FSLogger.Warning(msg)
			break
		}

		if extent.ExtentDataRem.LogicaAddress != 0 && extent.ExtentDataRem.LSize > 0 {
			buf.Write(hD.ReadFile(offset, int(extent.ExtentDataRem.LSize)))
			res := p.Sprintf("%d", (offset-partitionOffset)/int64(sectorsPerCluster*bytesPerSector))

			msg := fmt.Sprintf("offset %s cl len %d cl.", res, extent.ExtentDataRem.LSize)
			logger.FSLogger.Info(msg)
		}

	}

	//truncate buf grows over len?
	results <- utils.AskedFile{Fname: fileDirEntry.GetFname(), Content: buf.Bytes()[:lSize], Id: int(fileDirEntry.Id)}
}

func (fileDirEntry FileDirEntry) ShowAttributes(attrName string) {
	if attrName == "FileName" {
		fileDirEntry.ShowFileName()
	}

}

func (fileDirEntry FileDirEntry) ShowFileName() {
	fmt.Printf("%s\n", fileDirEntry.GetFname())

}

func (fileDirEntry FileDirEntry) ShowFileSize() {
	fmt.Printf("%d\n", fileDirEntry.GetLogicalFileSize())
}

func (fileDirEntry FileDirEntry) ShowIndex() {
	fmt.Printf("%d\n", fileDirEntry.GetIndex())
}

func (fileDirEntry FileDirEntry) ShowInfo() {
	fmt.Printf("%d ", fileDirEntry.Id)
}

func (fileDirEntry FileDirEntry) ShowIsResident() {

}

func (fileDirEntry FileDirEntry) ShowParentRecordInfo() {
	fileDirEntry.Parent.ShowInfo()

}

func (fileDirEntry FileDirEntry) ShowPath(pathtype int) {
	fmt.Printf("%s ", fileDirEntry.Path)
}

func (fileDirEntry FileDirEntry) ShowRunList() {
	for _, extent := range fileDirEntry.GetExtents() {
		fmt.Printf("%s \t", extent.GetInfo())
	}
}

func (fileDirEntry FileDirEntry) ShowTimestamps() {
	fmt.Printf("%s\n", fileDirEntry.GetTimestamps())
}

func (fileDirEntry FileDirEntry) ShowVCNs() {

}

func (fileDirEntry FileDirEntry) GetTimestamps() string {
	for _, attr := range fileDirEntry.FindAttributes("INODE_ITEM") {
		return attr.(*attributes.InodeItem).GetTimestamps()
	}
	return ""
}

func (fileDirEntry FileDirEntry) GetInfo() string {
	return ""
}

func (fileDirEntry FileDirEntry) GetExtentsInfo() []string {

	var extentsInfo []string
	for _, extent := range fileDirEntry.GetExtents() {
		extent.GetInfo()
	}
	return extentsInfo
}

func (fileDirEntry *FileDirEntry) BuildPath() {
	parent := fileDirEntry.Parent
	var paths []string
	for parent != nil {

		paths = append(paths, parent.GetFname())
		parent = parent.Parent
	}

	for idx := range paths {
		path := paths[len(paths)-idx-1]
		if path == "" {
			continue
		}
		fileDirEntry.Path += "\\" + path
	}

}

func (fileDirEntry FileDirEntry) FindAttribute(attrName string) leafnode.DataItem {
	return fileDirEntry.FindAttributes(attrName)[0]
}

func (fileDirEntry FileDirEntry) FindAttributes(attrName string) []leafnode.DataItem {
	var attributes []leafnode.DataItem
	for idx, attribute := range fileDirEntry.DataItems {
		if fileDirEntry.Items[idx].GetType() == attrName {
			attributes = append(attributes, attribute)
		}
	}
	return attributes
}

func (filesDirsMap FilesDirsMap) BuildPath() {
	for inodeId, fileDirEntry := range filesDirsMap {

		fileDirEntry.BuildPath()
		filesDirsMap[inodeId] = fileDirEntry
	}
}
