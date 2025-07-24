package volume

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"sync"

	metadata "github.com/aarsakian/FileSystemForensics/FS"
	fstree "github.com/aarsakian/FileSystemForensics/FS/BTRFS"
	"github.com/aarsakian/FileSystemForensics/FS/BTRFS/attributes"
	"github.com/aarsakian/FileSystemForensics/FS/BTRFS/leafnode"
	"github.com/aarsakian/FileSystemForensics/logger"
	"github.com/aarsakian/FileSystemForensics/readers"
	"github.com/aarsakian/FileSystemForensics/utils"
)

const SUPERBLOCKSIZE = 4096

const SYSTEMCHUNKARRSIZE = 2048

const OFFSET_TO_SUPERBLOCK = 0x10000 //64KB fixed position

const CHANNELSIZE = 10000

type BTRFS struct {
	Superblock   *Superblock
	Trees        []fstree.Tree
	ChunkTreeMap fstree.ChunkTreeMap
	FsTreeMap    fstree.FsTreeMap
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

func (btrfs BTRFS) GetSectorsPerCluster() int {
	return int(btrfs.Superblock.NodeSize)
}

func (btrfs BTRFS) GetBytesPerSector() uint64 {
	return uint64(btrfs.Superblock.SectorSize)
}

func (btrfs BTRFS) GetFS() []metadata.Record {
	var fileDirEntries []metadata.Record
	for _, fstree := range btrfs.FsTreeMap {
		for _, filedirEntry := range fstree.FilesDirsMap {
			temp := filedirEntry
			fileDirEntries = append(fileDirEntries, metadata.BTRFSRecord{&temp})
		}
	}
	return fileDirEntries
}

func (btrfs BTRFS) CollectUnallocated(readers.DiskReader, int64, chan<- []byte) {

}

func (btrfs BTRFS) GetLogicalToPhysicalMap() map[uint64]metadata.Chunk {
	mapper := make(map[uint64]metadata.Chunk)
	for keyOffset, chunkItem := range btrfs.ChunkTreeMap {
		tmp := chunkItem
		mapper[keyOffset] = metadata.BTRFSChunk{tmp}

	}
	return mapper
}

func (btrfs BTRFS) GetUnallocatedClusters() []int {
	return []int{}
}

func (btrfs BTRFS) GetSignature() string {
	return utils.Hexify(btrfs.Superblock.Signature[:])
}

func (btrfs BTRFS) GetInfo() string {
	return ""
}

func (btrfs *BTRFS) ParseSuperblock(data []byte) error {

	logger.FSLogger.Info("Parsing superblock")
	superblock := new(Superblock)
	utils.Unmarshal(data, superblock)

	if !superblock.Verify(data) {
		msg := "superblock chcksum failed"
		logger.FSLogger.Error(msg)
		fmt.Printf("%s \n", msg)
		return errors.New(msg)
	}
	btrfs.Superblock = superblock
	return nil

}

func (superblock Superblock) Verify(data []byte) bool {
	return utils.ToUint32(superblock.Chksum[:]) == utils.CalcCRC32(data[32:])
}

func (btrfs BTRFS) VerifySuperBlock(data []byte) bool {
	return btrfs.Superblock.Verify(data)
}

func (btrfs *BTRFS) Process(hD readers.DiskReader, partitionOffsetB int64, selectedEntries []int,
	fromEntry int, toEntry int) {

	btrfs.ChunkTreeMap = make(fstree.ChunkTreeMap)
	btrfs.UpdateChunkMaps(btrfs.ParseSystemChunks())

	verify := true
	node, err := btrfs.ParseTreeNode(hD, int(btrfs.Superblock.LogicalAddressRootChunkTree),
		partitionOffsetB,
		int(btrfs.Superblock.NodeSize), verify)
	if err != nil {

		logger.FSLogger.Error(err)
		log.Fatal(err)
	}

	btrfs.UpdateChunkMaps(node)

	/* parse root of root trees*/

	genericNodes, err := btrfs.ParseTreeNode(hD,
		int(btrfs.Superblock.LogicalAddressRootTree), partitionOffsetB,
		int(btrfs.Superblock.NodeSize), verify)
	if err != nil {

		logger.FSLogger.Error(err)
		log.Fatal(err)
	}

	subvolumename := ""
	btrfs.DiscoverTrees(genericNodes, subvolumename)
	carve := false
	btrfs.DescendTreeCh(hD, partitionOffsetB,
		int(btrfs.Superblock.NodeSize), verify, carve)

}

func (btrfs BTRFS) HasValidSignature() bool {
	return btrfs.GetSignature() == "5f42485266535f4d" //"_BHRfS_M"
}

func (btrfs *BTRFS) DescendTreeCh(hD readers.DiskReader, partitionOffsetB int64,
	nodeSize int, noverify bool, carve bool) {
	wg := new(sync.WaitGroup)

	for inodeId, fsTree := range btrfs.FsTreeMap {
		nodesLeaf := make(chan *fstree.GenericNode, CHANNELSIZE)
		logger.FSLogger.Info(fmt.Sprintf("Parsing tree %d", inodeId))

		wg.Add(2)
		go btrfs.ParseTreeNodeCh(hD, wg, int(fsTree.LogicalOffset), partitionOffsetB, nodeSize, nodesLeaf, noverify, carve)

		go btrfs.FsTreeMap.Update(wg, nodesLeaf, inodeId)

	}

	wg.Wait()

}

// returns leaf nodes
func (btrfs BTRFS) ParseTreeNodeCh(hD readers.DiskReader, wg *sync.WaitGroup, logicalOffset int,
	partitionOffsetB int64, size int, leafNodes chan<- *fstree.GenericNode,
	noverify bool, carve bool) {
	defer wg.Done()
	defer close(leafNodes)

	var internalNodes fstree.Stack

	node, err := btrfs.CreateNode(hD, logicalOffset, partitionOffsetB, size, noverify, carve)

	if err != nil {
		logger.FSLogger.Error(err)
		return
	}

	logger.FSLogger.Info(fmt.Sprintf("First node GUID %s chk %d ", node.GetGuid(), node.ChsumToUint()))

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

			logger.FSLogger.Info(fmt.Sprintf("Node GUID %s  from internal item id %d %d",
				node.GetGuid(), item.Key.ObjectID, parentCHK))

			if node.InternalNode != nil {
				internalNodes.Push(node)
			} else {
				leafNodes <- node

			}

		}

	}

}

/*ROOT_ITEMs: Metadata for each subvolume or snapshot

ROOT_REF and ROOT_BACKREF: Relationships between subvolumes

for each subvolume
INODE_ITEMs (file metadata)

DIR_ITEMs (directory entries)

DIR_INDEX_ITEMs (hashed directory entries for fast lookup)

INODE_REFs (reverse references to parent directories)

fs_tree per subvolume info the B-tree that stores the actual file and directory

root_dir_tree top-level subvolume always 1
contains only
INODE_ITEM, DIR_ITEM, INODE_REF
objectid = 2 → inode of the top-level directory
5->	Top-level subvolume ID contains DIR_ITEMs for all other subvolumes created
256+->	Subvolume/snapshot IDs

*/

func (btrfs *BTRFS) DiscoverTrees(nodes fstree.GenericNodesPtr, nametree string) {
	logger.FSLogger.Info("Parsing root of root trees")

	fsTrees := make(fstree.FsTreeMap)

	for _, node := range nodes {

		for idx, item := range node.LeafNode.Items {
			msg := fmt.Sprintf("Node %s item %s", node.GetGuid(), item.GetInfo())
			logger.FSLogger.Info(msg)

			if item.IsRootItem() {
				rootItem := node.LeafNode.DataItems[idx].(*attributes.RootItem)
				fsTree, ok := fsTrees[item.Key.ObjectID]
				if !ok {
					fsTrees[item.Key.ObjectID] = fstree.FsTree{
						LogicalOffset: rootItem.Bytenr,
						Uuid:          utils.StringifyUUID(rootItem.Uuid[:]),
					}
				} else {
					fsTree.LogicalOffset = rootItem.Bytenr
					fsTree.Uuid = utils.StringifyUUID(rootItem.Uuid[:])
					fsTrees[item.Key.ObjectID] = fsTree
				}

			} else if item.IsRootRef() {
				name := node.LeafNode.DataItems[idx].(*attributes.RootRef).Name

				fsTree, ok := fsTrees[item.Key.Offset]

				if !ok {
					fsTrees[item.Key.Offset] = fstree.FsTree{
						ParentId: item.Key.ObjectID,
						Name:     name,
					}
				} else {
					fsTree.ParentId = item.Key.ObjectID
					fsTree.Name = name
					fsTrees[item.Key.Offset] = fsTree
				}

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
func (btrfs BTRFS) CreateNode(hD readers.DiskReader, logicalOffset int, partitionOffsetB int64,
	size int, verify bool, carve bool) (*fstree.GenericNode, error) {

	physicalOffset, blockSize, err := btrfs.LocatePhysicalOffsetSize(uint64(logicalOffset))
	if err != nil {
		log.Fatal(err)
	}
	if int(blockSize) < size {
		size = int(blockSize)
	}
	data := hD.ReadFile(int64(physicalOffset)+partitionOffsetB, size)

	node := new(fstree.GenericNode)
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
	logger.FSLogger.Error(msg)
	return 0, 0, errors.New(msg)
}

// returns leaf nodes
func (btrfs BTRFS) ParseTreeNode(hD readers.DiskReader, logicalOffset int, partitionOffsetB int64,
	size int, verify bool) (fstree.GenericNodesPtr, error) {
	logger.FSLogger.Info(fmt.Sprintf("Parsing tree at %d", logicalOffset))

	var internalNodes fstree.Stack
	var leafNodes fstree.GenericNodesPtr
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

			logger.FSLogger.Info(fmt.Sprintf("created node from internal id %d %s nof items %d", item.Key.ObjectID, node.GetGuid(), node.Header.NofItems))
			if node.InternalNode != nil {
				internalNodes.Push(node)
			} else {
				leafNodes = append(leafNodes, node)

			}

		}

	}
	return leafNodes, nil

}

func (btrfs *BTRFS) UpdateChunkMaps(genericNodes fstree.GenericNodesPtr) {
	logger.FSLogger.Info("updating system chunks map")
	for _, node := range genericNodes {

		for idx, dataItem := range node.LeafNode.DataItems {
			if reflect.TypeOf(dataItem).Elem().Name() != "ChunkItem" {
				continue
			}
			chunkItem := dataItem.(*attributes.ChunkItem)
			key := node.LeafNode.Items[idx].Key

			btrfs.ChunkTreeMap[key.Offset] = chunkItem

		}

	}

}

func (btrfs BTRFS) ParseSystemChunks() fstree.GenericNodesPtr {
	// (KEY, CHUNK_ITEM) pairs for all SYSTEM chunks
	logger.FSLogger.Info("Parsing bootstrap data for system chunks.")

	curOffset := 0
	var genericNodes fstree.GenericNodesPtr
	data := btrfs.Superblock.SystemChunkArr[:btrfs.Superblock.SystemChunkArrSize]
	for curOffset < len(data) {
		key := new(attributes.Key)
		curOffset += key.Parse(data[curOffset:])

		chunkItem := new(attributes.ChunkItem)
		curOffset += chunkItem.Parse(data[curOffset:])

		leafNode := leafnode.LeafNode{}
		leafNode.Items = append(leafNode.Items, leafnode.Item{Key: key})
		leafNode.DataItems = append(leafNode.DataItems, chunkItem)
		node := fstree.GenericNode{LeafNode: &leafNode}

		genericNodes = append(genericNodes, &node)

	}
	return genericNodes
}

func (btrfs BTRFS) GetFSOffset() int64 {
	return 0
}
