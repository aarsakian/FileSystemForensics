package fstree

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aarsakian/FileSystemForensics/FS/BTRFS/attributes"
	"github.com/aarsakian/FileSystemForensics/FS/BTRFS/leafnode"
	"github.com/aarsakian/FileSystemForensics/logger"
	"github.com/aarsakian/FileSystemForensics/readers"
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

func (fileDirEntry FileDirEntry) GetExtentsMap() map[uint64]*attributes.ExtentData {
	//Map of logical offset in the file to extent
	extentsMap := make(map[uint64]*attributes.ExtentData)
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

func (fileDirEntry FileDirEntry) GetExtents() []*attributes.ExtentData {
	var extents []*attributes.ExtentData
	for idx, item := range fileDirEntry.Items {
		if !item.IsExtentData() {
			continue
		}
		extData, ok := fileDirEntry.DataItems[idx].(*attributes.ExtentData)
		if !ok {
			continue
		}
		extents = append(extents, extData)
	}
	return extents
}

func (fileDirEntry FileDirEntry) GetGroupedExtents() [][]*attributes.ExtentData {
	var groupedExtents [][]*attributes.ExtentData
	var extents []*attributes.ExtentData

	extentsMap := fileDirEntry.GetExtentsMap()
	logicalOffsetsInfile := make([]int, 0, len(extentsMap))

	for logicalOffsetInFile := range extentsMap {
		logicalOffsetsInfile = append(logicalOffsetsInfile, int(logicalOffsetInFile))
	}
	sort.Ints(logicalOffsetsInfile)

	for idx := range logicalOffsetsInfile {

		if idx == 0 || extentsMap[uint64(logicalOffsetsInfile[idx-1])].ExtentDataRem.LogicalAddress+
			extentsMap[uint64(logicalOffsetsInfile[idx-1])].ExtentDataRem.LSize ==
			extentsMap[uint64(logicalOffsetsInfile[idx])].ExtentDataRem.LogicalAddress &&
			extentsMap[uint64(logicalOffsetsInfile[idx])].ExtentDataRem.Offset == 0 &&
			extentsMap[uint64(logicalOffsetsInfile[idx])].ExtentDataRem.LogicalAddress != 0 {

			extents = append(extents, extentsMap[uint64(logicalOffsetsInfile[idx])])
		} else {
			groupedExtents = append(groupedExtents, extents)
			extents = []*attributes.ExtentData{extentsMap[uint64(logicalOffsetsInfile[idx])]}
		}
	}

	//case for one item
	if len(extents) > 0 {
		groupedExtents = append(groupedExtents, extents)
	}
	return groupedExtents

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

func (fileDirEntry FileDirEntry) GetParentID() int {
	return fileDirEntry.Parent.Id
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

func (fileDirEntry FileDirEntry) IsBase() bool {
	return true
}

func (fileDirEntry FileDirEntry) GetIndex() int {
	attr := fileDirEntry.FindAttributes("INODE_REF")[0].(*attributes.InodeRef)
	return int(attr.Index)
}

func (fileDirEntry FileDirEntry) LocateDataAsync(hD readers.DiskReader, partitionOffset int64,
	clusterSizeB int, dataFragments chan<- []byte) {

}

func (fileDirEntry FileDirEntry) ShowAttributes(attrName string) {
	if attrName == "FileName" {
		fileDirEntry.ShowFileName()
	}

}

func (fileDirEntry FileDirEntry) ShowAllocatedClusters() {

}

func (fileDirEntry FileDirEntry) ShowFileName() {
	fmt.Printf("%s\n", fileDirEntry.GetFname())

}

func (fileDirEntry FileDirEntry) ShowFileSize() {
	fmt.Printf("logical size %.1f KB\n",
		float64(fileDirEntry.GetLogicalFileSize()/1024))
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
	fmt.Printf("%s \n", filepath.Join(fileDirEntry.Path, fileDirEntry.GetFname()))
}

func (fileDirEntry FileDirEntry) ShowRunList() {
	for _, extent := range fileDirEntry.GetExtentsMap() {
		fmt.Printf("%s \n", extent.GetInfo())
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

func (fileDirEntry FileDirEntry) GetExtentsMapInfo() []string {

	var extentsInfo []string
	for _, extent := range fileDirEntry.GetExtentsMap() {
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
