package fstree

import (
	"sync"

	"github.com/aarsakian/FileSystemForensics/FS/BTRFS/attributes"
)

type ChunkTreeMap = map[uint64]*attributes.ChunkItem

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

				fileDirEntry := FileDirEntry{Id: int(item.Key.ObjectID)}
				fileDirEntry.Items = append(fileDirEntry.Items, item)
				fileDirEntry.DataItems = append(fileDirEntry.DataItems,
					node.LeafNode.DataItems[idx])

				fstree.FilesDirsMap[item.Key.ObjectID] = fileDirEntry

			} else if item.IsDirItem() {

				fileDirEntry := fstree.FilesDirsMap[item.Key.ObjectID]
				fileDirEntry.Items = append(fileDirEntry.Items, item)
				fileDirEntry.DataItems = append(fileDirEntry.DataItems,
					node.LeafNode.DataItems[idx])

				fstree.FilesDirsMap[item.Key.ObjectID] = fileDirEntry

			} else if item.IsDirIndex() {

				fileDirEntry := fstree.FilesDirsMap[item.Key.ObjectID]
				fileDirEntry.Index = int(item.Key.Offset)
				fileDirEntry.Items = append(fileDirEntry.Items, item)
				fileDirEntry.DataItems = append(fileDirEntry.DataItems,
					node.LeafNode.DataItems[idx])

				fstree.FilesDirsMap[item.Key.ObjectID] = fileDirEntry

			} else if item.IsInodeRef() {

				fileDirEntry := fstree.FilesDirsMap[item.Key.ObjectID]

				//obtains a copy
				fileDirParent := fstree.FilesDirsMap[item.Key.Offset]

				fileDirEntry.Parent = &fileDirParent

				fileDirParent.Children = append(fileDirParent.Children, &fileDirEntry)
				fileDirEntry.Items = append(fileDirEntry.Items, item)
				fileDirEntry.DataItems = append(fileDirEntry.DataItems,
					node.LeafNode.DataItems[idx])
				//reassign
				fstree.FilesDirsMap[item.Key.Offset] = fileDirParent
				fstree.FilesDirsMap[item.Key.ObjectID] = fileDirEntry

			} else if item.IsExtentData() {

				fileDirEntry := fstree.FilesDirsMap[item.Key.ObjectID]
				fileDirEntry.Items = append(fileDirEntry.Items, item)
				fileDirEntry.DataItems = append(fileDirEntry.DataItems,
					node.LeafNode.DataItems[idx])
				fstree.FilesDirsMap[item.Key.ObjectID] = fileDirEntry

			}

		}

	}

	fstree.FilesDirsMap.BuildPath()
	fsTreeMap[inodeid] = fstree
}
