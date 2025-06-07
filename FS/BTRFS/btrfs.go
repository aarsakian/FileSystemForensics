package BTRFS

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"sync"

	"github.com/aarsakian/FileSystemForensics/FS/BTRFS/leafnode"
	"github.com/aarsakian/FileSystemForensics/img"
	"github.com/aarsakian/FileSystemForensics/logger"
	"github.com/aarsakian/FileSystemForensics/utils"
)

const SYSTEMCHUNKARRSIZE = 2048
const SUPERBLOCKSIZE = 4096

const CHANNELSIZE = 10000

const OFFSET_TO_SUPERBLOCK = 0x10000 //64KB fixed position

type ChunkTreeMap = map[uint64]*leafnode.ChunkItem

type BTRFS struct {
	Superblock   *Superblock
	Trees        []Tree
	ChunkTreeMap ChunkTreeMap
	FsTreeMap    FsTreeMap
}

type Stack struct {
	nodes []*GenericNode
}

func (stack *Stack) Push(node *GenericNode) {
	stack.nodes = append(stack.nodes, node)
}

func (stack *Stack) Pop() *GenericNode {
	topNode := stack.nodes[len(stack.nodes)-1]
	stack.nodes = stack.nodes[:len(stack.nodes)-1]
	return topNode
}

func (stack Stack) IsEmpty() bool {
	return len(stack.nodes) == 0
}

type Tree struct {
	ParentId      uint64
	Name          string
	LogicalOffset uint64
	Nodes         GenericNodesPtr //low level slice of nodes
	Uuid          string
	FilesDirsMap  FilesDirsMap //map for files and folders per inodeid
}

type Superblock struct { //4096B
	Chksum                      [32]byte
	UUID                        [16]byte
	PhysicalAddress             uint64
	Flags                       uint64
	Signature                   [8]byte //8bytes 64 offset
	Generation                  uint64
	LogicalAddressRootTree      uint64
	LogicalAddressRootChunkTree uint64
	LogicalAddressLogRootTree   uint64
	LogRootTransid              uint64
	VolumeSizeB                 uint64
	VolumeUsedSizeB             uint64
	ObjectID                    uint64
	NofVolumeDevices            uint64
	SectorSize                  uint32
	NodeSize                    uint32
	LeafSize                    uint32
	StripeSize                  uint32
	SystemChunkArrSize          uint32
	ChunkRootGeneration         uint64
	CompatFlags                 uint64
	CompatROFlags               uint64
	InCompatFlags               uint64
	CsumType                    uint16
	RootLevel                   uint8 //198
	ChunkRootLevel              uint8
	LogRootLevel                uint8 //200
	DevItemArea                 [98]byte
	Label                       [256]byte //299
	Reserved                    [256]byte
	SystemChunkArr              [SYSTEMCHUNKARRSIZE]byte //811

}

func (btrfs *BTRFS) Process(hD img.DiskReader, partitionOffsetB int64, selectedEntries []int,
	fromEntry int, toEntry int) {

	msg := "Reading superblock at offset %d"
	fmt.Printf(msg+"\n", partitionOffsetB)
	logger.MFTExtractorlogger.Info(fmt.Sprintf(msg, partitionOffsetB))

	data := hD.ReadFile(partitionOffsetB+OFFSET_TO_SUPERBLOCK, SUPERBLOCKSIZE)
	btrfs.ParseSuperblock(data)
	if !btrfs.VerifySuperBlock(data) {
		msg := "superblock chcksum failed!"
		logger.MFTExtractorlogger.Error(msg)
		return
	}
	btrfs.ChunkTreeMap = make(ChunkTreeMap)
	btrfs.UpdateChunkMaps(btrfs.ParseSystemChunks())

	verify := true
	node, err := btrfs.ParseTreeNode(hD, int(btrfs.Superblock.LogicalAddressRootChunkTree),
		partitionOffsetB,
		int(btrfs.Superblock.NodeSize), verify)
	if err != nil {

		logger.MFTExtractorlogger.Error(err)
		log.Fatal(err)
	}

	btrfs.UpdateChunkMaps(node)

	/* parse root of root trees*/

	genericNodes, err := btrfs.ParseTreeNode(hD, int(btrfs.Superblock.LogicalAddressRootTree), partitionOffsetB,
		int(btrfs.Superblock.NodeSize), verify)
	if err != nil {

		logger.MFTExtractorlogger.Error(err)
		log.Fatal(err)
	}

	subvolumename := ""
	btrfs.DiscoverTrees(genericNodes, subvolumename)
	carve := false
	btrfs.DescendTreeCh(hD, partitionOffsetB, int(btrfs.Superblock.NodeSize), verify, carve)

}

func (btrfs *BTRFS) DescendTreeCh(hD img.DiskReader, partitionOffsetB int64,
	nodeSize int, noverify bool, carve bool) {
	wg := new(sync.WaitGroup)

	for inodeId, fsTree := range btrfs.FsTreeMap {
		nodesLeaf := make(chan *GenericNode, CHANNELSIZE)
		logger.MFTExtractorlogger.Info(fmt.Sprintf("Parsing tree %d", inodeId))

		wg.Add(2)
		go btrfs.ParseTreeNodeCh(hD, wg, int(fsTree.LogicalOffset), partitionOffsetB, nodeSize, nodesLeaf, noverify, carve)

		go btrfs.FsTreeMap.Update(wg, nodesLeaf, inodeId)

	}

	wg.Wait()

}

func (fsTreeMap FsTreeMap) Update(wgs *sync.WaitGroup, nodesLeaf chan *GenericNode, inodeid uint64) {
	defer wgs.Done()

	fstree := fsTreeMap[inodeid]
	if fstree.FilesDirsMap == nil {
		fstree.FilesDirsMap = make(FilesDirsMap)
	}

	for node := range nodesLeaf {

		fstree.Nodes = append(fstree.Nodes, node)

		for idx, item := range node.LeafNode.Items {

			if item.IsInodeItem() {

				dataItem := node.LeafNode.DataItems[idx].(*leafnode.InodeItem)
				fstree.FilesDirsMap[item.Key.ObjectID] = FileDirEntry{Id: int(item.Key.ObjectID),
					SizeB: int(dataItem.StSize), Nlink: int(dataItem.StNlink),
					Uid: int(dataItem.StUid), Gid: int(dataItem.StGid), ATime: dataItem.ATime.ToTime(),
					MTime: dataItem.MTime.ToTime(),
					CTime: dataItem.CTime.ToTime(), OTime: dataItem.OTime.ToTime(), Type: dataItem.GetType()}

			} else if item.IsDirItem() {

				fileDirEntry := fstree.FilesDirsMap[item.Key.ObjectID]

				dataItem := node.LeafNode.DataItems[idx].(*leafnode.DirItem)
				fileDirEntry.Flags = dataItem.GetType()
				fileDirEntry.Type = dataItem.GetType()

				fstree.FilesDirsMap[item.Key.ObjectID] = fileDirEntry

			} else if item.IsDirIndex() {

				fileDirEntry := fstree.FilesDirsMap[item.Key.ObjectID]
				fileDirEntry.Index = int(item.Key.Offset)

				fstree.FilesDirsMap[item.Key.ObjectID] = fileDirEntry

			} else if item.IsInodeRef() {

				fileDirEntry := fstree.FilesDirsMap[item.Key.ObjectID]

				dataItem := node.LeafNode.DataItems[idx].(*leafnode.InodeRef)
				//obtains a copy
				fileDirParent := fstree.FilesDirsMap[item.Key.Offset]

				fileDirEntry.Index = int(dataItem.Index)
				fileDirEntry.Parent = &fileDirParent
				fileDirEntry.Name = dataItem.Name

				fileDirParent.Children = append(fileDirParent.Children, &fileDirEntry)

				//reassign
				fstree.FilesDirsMap[item.Key.Offset] = fileDirParent
				fstree.FilesDirsMap[item.Key.ObjectID] = fileDirEntry

			} else if item.IsExtentData() {

				fileDirEntry := fstree.FilesDirsMap[item.Key.ObjectID]

				dataItem := node.LeafNode.DataItems[idx].(*leafnode.ExtentData)
				if dataItem.GetType() == "Inline Extent" {

				} else if dataItem.GetType() == "Regular Extent" {
					extent := Extent{Offset: int(dataItem.ExtentDataRem.LogicaAddress),
						PSize: int(dataItem.ExtentDataRem.Size), LSize: int(dataItem.ExtentDataRem.LogicalBytes)}
					fileDirEntry.Extents = append(fileDirEntry.Extents, extent)
					fstree.FilesDirsMap[item.Key.ObjectID] = fileDirEntry
				}

			}

		}

	}
	fsTreeMap[inodeid] = fstree
}

// returns leaf nodes
func (btrfs BTRFS) ParseTreeNodeCh(hD img.DiskReader, wg *sync.WaitGroup, logicalOffset int,
	partitionOffsetB int64, size int, leafNodes chan<- *GenericNode,
	noverify bool, carve bool) {
	defer wg.Done()
	defer close(leafNodes)

	var internalNodes Stack

	node, err := btrfs.CreateNode(hD, logicalOffset, partitionOffsetB, size, noverify, carve)

	if err != nil {
		logger.MFTExtractorlogger.Error(err)
		return
	}

	logger.MFTExtractorlogger.Info(fmt.Sprintf("First node GUID %s chk %d ", node.GetGuid(), node.ChsumToUint()))

	if node.InternalNode != nil {
		internalNodes.Push(node)
	} else {
		//	fmt.Println("sending no internal node", logicalOffset)
		leafNodes <- node
	}

	for !internalNodes.IsEmpty() {
		node := internalNodes.Pop()
		parentCHK := node.ChsumToUint()

		for _, item := range node.InternalNode.Items {
			node, err := btrfs.CreateNode(hD, int(item.LogicalAddressRefHeader), partitionOffsetB, size, noverify, carve)
			if err != nil {
				continue
			}

			logger.MFTExtractorlogger.Info(fmt.Sprintf("Node GUID %s  from internal item id %d %d",
				node.GetGuid(), item.Key.ObjectID, parentCHK))

			if node.InternalNode != nil {
				internalNodes.Push(node)
			} else {
				leafNodes <- node

			}

		}

	}

}

func (btrfs *BTRFS) DiscoverTrees(nodes GenericNodesPtr, nametree string) {
	logger.MFTExtractorlogger.Info("Parsing root of root trees")

	var dir DirTree
	fsTrees := make(FsTreeMap)

	for _, node := range nodes {

		for idx, item := range node.LeafNode.Items {
			msg := fmt.Sprintf("Node %s item  obj %d offs %d Type %s  %s", node.GetGuid(), item.Key.ObjectID, item.Key.Offset,
				leafnode.ItemTypes[int(item.Key.ItemType)], leafnode.ObjectTypes[int(item.Key.ObjectID)])
			logger.MFTExtractorlogger.Info(msg)

			if item.IsRootItem() {
				rootItem := node.LeafNode.DataItems[idx].(*leafnode.RootItem)
				fsTree, ok := fsTrees[item.Key.ObjectID]
				if !ok {
					fsTrees[item.Key.ObjectID] = FsTree{
						LogicalOffset: rootItem.Bytenr,
						Uuid:          utils.StringifyGUID(rootItem.Uuid[:]),
					}
				} else {
					fsTree.LogicalOffset = rootItem.Bytenr
					fsTree.Uuid = utils.StringifyGUID(rootItem.Uuid[:])
					fsTrees[item.Key.ObjectID] = fsTree
				}

			} else if item.IsRootRef() {
				name := node.LeafNode.DataItems[idx].(*leafnode.RootRef).Name

				fsTree, ok := fsTrees[item.Key.Offset]

				if !ok {
					fsTrees[item.Key.Offset] = FsTree{
						ParentId: item.Key.ObjectID,
						Name:     name,
					}
				} else {
					fsTree.ParentId = item.Key.ObjectID
					fsTree.Name = name
					fsTrees[item.Key.Offset] = fsTree
				}

			} else if item.IsInodeItem() && item.IsDIRTree() {
				inode := node.LeafNode.DataItems[idx].(*leafnode.InodeItem)
				dir = DirTree{Id: item.Key.ObjectID, InodePtr: inode}

			} else if item.IsDirItem() && item.IsDIRTree() {
				dir.Name = node.LeafNode.DataItems[idx].(*leafnode.DirItem).Name
			} else if item.IsInodeRef() && item.IsDIRTree() {
				dir.Index = node.LeafNode.DataItems[idx].(*leafnode.InodeRef).Index
			}
		}

	}

	if nametree != "" {

		for id, fsTree := range fsTrees {
			if fsTree.Name != nametree {
				delete(fsTrees, id)

			}
		}

	}
	btrfs.FsTreeMap = fsTrees
}

// producer of nodes
func (btrfs BTRFS) CreateNode(hD img.DiskReader, logicalOffset int, partitionOffsetB int64,
	size int, verify bool, carve bool) (*GenericNode, error) {

	physicalOffset, blockSize, err := btrfs.LocatePhysicalOffsetSize(uint64(logicalOffset))
	if err != nil {
		log.Fatal(err)
	}
	if int(blockSize) < size {
		size = int(blockSize)
	}
	data := hD.ReadFile(int64(physicalOffset)+partitionOffsetB, size)

	node := new(GenericNode)
	_, err = node.Parse(data, int64(physicalOffset)+partitionOffsetB, verify, carve)
	if err != nil {
		return nil, err
	}

	return node, nil

}

func (btrfs BTRFS) LocatePhysicalOffsetSize(logicalOffset uint64) (uint64, uint64, error) {
	for keyOffset, chunkItem := range btrfs.ChunkTreeMap {
		if logicalOffset >= keyOffset && logicalOffset-keyOffset < chunkItem.Size {
			blockGroupOffset := logicalOffset - keyOffset
			return chunkItem.Stripes[0].Offset + blockGroupOffset, chunkItem.Size, nil
		}
	}
	msg := fmt.Sprintf("physical offset not located for logical offset %d", logicalOffset)
	logger.MFTExtractorlogger.Error(msg)
	return 0, 0, errors.New(msg)
}

// returns leaf nodes
func (btrfs BTRFS) ParseTreeNode(hD img.DiskReader, logicalOffset int, partitionOffsetB int64,
	size int, verify bool) (GenericNodesPtr, error) {
	logger.MFTExtractorlogger.Info(fmt.Sprintf("Parsing tree at %d", logicalOffset))

	var internalNodes Stack
	var leafNodes GenericNodesPtr
	carve := false

	node, err := btrfs.CreateNode(hD, logicalOffset, partitionOffsetB, size, verify, carve)

	if err != nil {
		return nil, err
	}

	if node.InternalNode != nil {
		internalNodes.Push(node)
	} else {
		leafNodes = append(leafNodes, node)
	}

	for !internalNodes.IsEmpty() {
		node := internalNodes.Pop()

		for _, item := range node.InternalNode.Items {

			node, err := btrfs.CreateNode(hD, int(item.LogicalAddressRefHeader), partitionOffsetB, size, verify, carve)
			if err != nil {
				continue
			}

			logger.MFTExtractorlogger.Info(fmt.Sprintf("created node from internal id %d %s nof items %d", item.Key.ObjectID, node.GetGuid(), node.Header.NofItems))
			if node.InternalNode != nil {
				internalNodes.Push(node)
			} else {
				leafNodes = append(leafNodes, node)

			}

		}

	}
	return leafNodes, nil

}

func (btrfs *BTRFS) UpdateChunkMaps(genericNodes GenericNodesPtr) {
	logger.MFTExtractorlogger.Info("updating system chunks map")
	for _, node := range genericNodes {

		for idx, dataItem := range node.LeafNode.DataItems {
			if reflect.TypeOf(dataItem).Elem().Name() != "ChunkItem" {
				continue
			}
			chunkItem := dataItem.(*leafnode.ChunkItem)
			key := node.LeafNode.Items[idx].Key

			btrfs.ChunkTreeMap[key.Offset] = chunkItem

		}

	}

}

func (btrfs *BTRFS) ParseSuperblock(data []byte) {
	logger.MFTExtractorlogger.Info("Parsing superblock")
	superblock := new(Superblock)
	utils.Unmarshal(data, superblock)

	btrfs.Superblock = superblock

}

func (btrfs BTRFS) VerifySuperBlock(data []byte) bool {
	return utils.ToUint32(btrfs.Superblock.Chksum[:]) == utils.CalcCRC32(data[32:])

}

func (btrfs BTRFS) ParseSystemChunks() GenericNodesPtr {
	// (KEY, CHUNK_ITEM) pairs for all SYSTEM chunks
	logger.MFTExtractorlogger.Info("Parsing bootstrap data for system chunks.")

	curOffset := 0
	var genericNodes GenericNodesPtr
	data := btrfs.Superblock.SystemChunkArr[:btrfs.Superblock.SystemChunkArrSize]
	for curOffset < len(data) {
		key := new(leafnode.Key)
		curOffset += key.Parse(data[curOffset:])

		chunkItem := new(leafnode.ChunkItem)
		curOffset += chunkItem.Parse(data[curOffset:])

		leafNode := leafnode.LeafNode{}
		leafNode.Items = append(leafNode.Items, leafnode.Item{Key: key})
		leafNode.DataItems = append(leafNode.DataItems, chunkItem)
		node := GenericNode{LeafNode: &leafNode}

		genericNodes = append(genericNodes, &node)

	}
	return genericNodes
}

func (btrfs BTRFS) GetSectorsPerCluster() int {
	return int(btrfs.Superblock.NodeSize)
}

func (btrfs BTRFS) GetBytesPerSector() uint64 {
	return uint64(btrfs.Superblock.SectorSize)
}

func (btrfs BTRFS) GetMetadata() []FileDirEntry {
	//return btrfs.tree.fileDirEntries
	return nil
}

func (btrfs BTRFS) CollectUnallocated(img.DiskReader, int64, chan<- []byte) {

}

func (btrfs BTRFS) GetSignature() string {
	return string(btrfs.Superblock.Signature[:])
}
