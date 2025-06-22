package BTRFS

import (
	"errors"
	"fmt"
	"strconv"

	//exporter "github.com/aarsakian/FileSystemForensics/Exporter"
	"github.com/aarsakian/FileSystemForensics/FS/BTRFS/internalnode"
	"github.com/aarsakian/FileSystemForensics/FS/BTRFS/leafnode"
	"github.com/aarsakian/FileSystemForensics/logger"
	"github.com/aarsakian/FileSystemForensics/utils"
)

type FsTreeMap map[uint64]FsTree

// core structure for each root tree
type FsTree struct {
	ParentId      uint64
	Name          string
	LogicalOffset uint64
	Nodes         GenericNodesPtr //low level slice of nodes
	Uuid          string
	FilesDirsMap  FilesDirsMap //map for files and folders per inodeid
}

type DirTree struct {
	Id       uint64
	Name     string
	InodePtr *leafnode.InodeItem
	Index    uint64
}

type Key struct {
	ObjectID uint64
	ItemType uint8
	Offset   uint64
}

type Header struct { //101Bytes
	Chksum             [32]byte
	FsUUID             [16]byte
	LogicalAddressNode uint64
	Flags              [7]byte
	BackRef            uint8
	ChunkTreeUUID      [16]byte
	Generation         uint64
	Owner              uint64
	NofItems           uint32
	Level              uint8
}

type GenericNodesMap map[uint64][]GenericNodesPtr

type GenericNodes []GenericNode
type GenericNodesPtr []*GenericNode

type GenericNode struct {
	Header       *Header
	LeafNode     *leafnode.LeafNode
	InternalNode *internalnode.InternalNode
}

func (key *Key) Parse(data []byte) int {

	offset, _ := utils.Unmarshal(data, key)
	return offset
}

func (nodesPtr GenericNodesPtr) FilterItemsByIds(ids []string) ([]leafnode.Item,
	[]leafnode.DataItem) {

	var items []leafnode.Item
	var dataItems []leafnode.DataItem

	for _, id := range ids {
		val, err := strconv.ParseUint(id, 10, 64)
		if err != nil {
			continue
		}
		for _, node := range nodesPtr {
			nitems, ndataItems := node.FilterItemsById(val)
			items = append(items, nitems...)
			dataItems = append(dataItems, ndataItems...)

		}
	}
	return items, dataItems
}

/*func (nodesPtr GenericNodesPtr) WriteFilesDirsInfo(exp exporter.Exporter) {

		for _, node := range nodesPtr {
		for idx, item := range node.LeafNode.Items {
			if item.IsDirItem() {
				dirItem := node.LeafNode.DataItems[idx].(*leafnode.DirItem)
				if dirItem.GetType() == "BTRFS_TYPE_FILE" || dirItem.GetType() == "BTRFS_TYPE_DIRECTORY" {
					exp.WriteFile(fmt.Sprintf("Dir Inode %d index id %d ", item.Key.ObjectID, item.Key.Offset))
					exp.WriteFile(dirItem.GetInfo())
				}
			} else if item.IsDirIndex() {
				dirIdx := node.LeafNode.DataItems[idx].(*leafnode.DirIndex)
				if dirIdx.GetType() == "BTRFS_TYPE_FILE" || dirIdx.GetType() == "BTRFS_TYPE_DIRECTORY" {
					exp.WriteFile(fmt.Sprintf("Dir Idx %d ", item.Key.ObjectID))
					exp.WriteFile(node.LeafNode.DataItems[idx].GetInfo())
				}

			} else if item.IsInodeItem() {
				exp.WriteFile(fmt.Sprintf("Inode %d ", item.Key.ObjectID))
				exp.WriteFile(node.LeafNode.DataItems[idx].GetInfo())
			} else if item.IsInodeRef() {
				exp.WriteFile(fmt.Sprintf("Inode ref %d Parent Inode %d ", item.Key.ObjectID, item.Key.Offset))
				exp.WriteFile(node.LeafNode.DataItems[idx].GetInfo())
			}

		}

	}

}*/

func (nodesPtr GenericNodesPtr) ShowFilesDirsInfo() {

	for _, node := range nodesPtr {
		for idx, item := range node.LeafNode.Items {
			if item.IsDirItem() {
				dirItem := node.LeafNode.DataItems[idx].(*leafnode.DirItem)
				if dirItem.GetType() == "BTRFS_TYPE_FILE" || dirItem.GetType() == "BTRFS_TYPE_DIRECTORY" {
					fmt.Printf("Dir Inode %d index id %d %s", item.Key.ObjectID, item.Key.Offset, dirItem.GetInfo())

				}
			} else if item.IsDirIndex() {
				dirIdx := node.LeafNode.DataItems[idx].(*leafnode.DirIndex)
				if dirIdx.GetType() == "BTRFS_TYPE_FILE" || dirIdx.GetType() == "BTRFS_TYPE_DIRECTORY" {
					fmt.Printf("Dir Idx %d %s", item.Key.ObjectID, node.LeafNode.DataItems[idx].GetInfo())

				}

			} else if item.IsInodeItem() {
				fmt.Printf("Inode %d %s", item.Key.ObjectID, node.LeafNode.DataItems[idx].GetInfo())

			} else if item.IsInodeRef() {
				fmt.Printf("Inode ref %d Parent Inode %d %s", item.Key.ObjectID, item.Key.Offset, node.LeafNode.DataItems[idx].GetInfo())

			}

		}

	}

}

func (genericNode GenericNode) FilterItemsById(id uint64) ([]leafnode.Item, []leafnode.DataItem) {
	var items []leafnode.Item
	var dataItems []leafnode.DataItem
	if genericNode.LeafNode != nil {
		for idx, item := range genericNode.LeafNode.Items {
			if item.Key.ObjectID != id {
				continue
			}
			items = append(items, item)
			dataItems = append(dataItems, genericNode.LeafNode.DataItems[idx])
		}
	}
	return items, dataItems
}

func (genericNode GenericNode) ChsumToUint() uint32 {
	return utils.ToUint32(genericNode.Header.Chksum[:])
}

func (genericNode GenericNode) VerifyChkSum(data []byte) bool {
	return genericNode.ChsumToUint() == utils.CalcCRC32(data[32:])
}

func (genericNode GenericNode) GetGuid() string {
	return utils.StringifyGUID(genericNode.Header.ChunkTreeUUID[:])
}

func (genericNode *GenericNode) Parse(data []byte, physicalOffset int64, verify bool, carve bool) (int, error) {
	offset := 0
	genericNode.Header = new(Header)
	offset, _ = utils.Unmarshal(data, genericNode.Header)

	if verify && !genericNode.VerifyChkSum(data) {

		msg := fmt.Sprintf("Node verification failed at %d", physicalOffset)
		logger.FSLogger.Error(msg)
		return -1, errors.New(msg)
	} else if verify {
		logger.FSLogger.Info(fmt.Sprintf("Node verification sucess at %d level %d items %d",
			physicalOffset, genericNode.Header.Level, genericNode.Header.NofItems))
	} else {
		logger.FSLogger.Info(fmt.Sprintf("Node at %d level %d items %d",
			physicalOffset, genericNode.Header.Level, genericNode.Header.NofItems))
	}

	if genericNode.Header.Level == 0 { //leaf Node
		leafNode := new(leafnode.LeafNode)
		leafNode.Items = make([]leafnode.Item, genericNode.Header.NofItems)
		leafNode.DataItems = make([]leafnode.DataItem, genericNode.Header.NofItems)

		offset += leafNode.Parse(data[offset:], physicalOffset+int64(offset))
		genericNode.LeafNode = leafNode

	} else {
		internalNode := new(internalnode.InternalNode)
		internalNode.Items = make([]internalnode.BlockPointer, genericNode.Header.NofItems)
		offset += internalNode.Parse(data[offset:], physicalOffset+int64(offset))
		genericNode.InternalNode = internalNode
	}

	return offset, nil
}

func (key Key) ShowInfo() {
	fmt.Printf("key %s  %s %d\n", leafnode.ItemTypes[int(key.ItemType)],
		leafnode.ObjectTypes[int(key.ObjectID)], key.Offset)
}

func (nodes GenericNodesPtr) ShowInfo() {
	fmt.Printf("\n")
	for _, node := range nodes {
		node.ShowInfo()
	}
}

func (node GenericNode) ShowInfo() {
	if node.LeafNode != nil {
		node.Header.ShowInfo()
		node.LeafNode.ShowInfo()
	}
}

func (header Header) ShowInfo() {
	fmt.Printf("level %d bytenr %d nritems %d\n", header.Level, header.LogicalAddressNode, header.NofItems)
}
