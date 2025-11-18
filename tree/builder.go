package tree

import (
	"fmt"
	"sort"
	"strings"

	metadata "github.com/aarsakian/FileSystemForensics/FS"
	"github.com/aarsakian/FileSystemForensics/logger"
)

/*Thus, a B-tree node is equivalent to a disk block, and a “pointer” value stored
in the tree is actually the number of the block containing the child node (usually
interpreted as an offset from the beginning of the corresponding disk file)*/

type Node struct {
	ID       int
	ParentID int
	record   *metadata.Record
	parent   *Node
	children []*Node
}

type Tree struct {
	root          *Node
	OrphanedNodes []*Node
}

func (t *Tree) Build(records []metadata.Record) {
	msg := fmt.Sprintf("Building tree from %d MFT records ", len(records))
	fmt.Printf(msg + "\n")
	logger.FSLogger.Info(msg)
	nodeMap := make(map[int]*Node)

	for idx := range records {
		if records[idx].GetID() < 5 { //$MFT entry number 5
			continue
		}

		//skip non base records
		if !records[idx].IsBase() {
			continue
		}
		parentID := records[idx].GetParentID()

		if parentID == -1 {
			continue
		}

		nodeMap[records[idx].GetID()] = &Node{records[idx].GetID(), parentID,
			&records[idx], nil, nil}
		//t.AddRecord(&records[idx])
	}

	for _, node := range nodeMap {
		if node.ID == 5 {
			t.root = node
			continue
		}

		parent, ok := nodeMap[node.ParentID]
		if ok {
			parent.children = append(parent.children, node)
			node.parent = parent
		} else {
			// Orphaned node
			t.OrphanedNodes = append(t.OrphanedNodes, node)
		}

	}

}

func (t Tree) Show() {

	t.root.descend()
	t.showOrphanedNodes()

}

func (node Node) descend() {
	if node.children == nil {
		return

	}
	node.showChildrenInfo()
	for _, node := range node.children {

		node.descend()

	}
}

func (t Tree) showOrphanedNodes() {
	if t.OrphanedNodes == nil {
		return
	}
	fmt.Print("\n Orphaned Nodes: \n")
	for _, orphanedNode := range t.OrphanedNodes {
		orphanedNode.showChildrenInfo()
	}
}

func (node Node) showChildrenInfo() {
	msgB := strings.Builder{}
	msgB.Grow(len(node.children) + 1) // for root
	if (*node.record).IsFolder() {
		msg := fmt.Sprintf(" %s  (%d) |_> ", (*node.record).GetFname(), (*node.record).GetID())

		msgB.WriteString(msg)

		fmt.Print("\n" + msg)

		sort.Slice(node.children, func(i, j int) bool {
			return (*node.children[i].record).GetFname() < (*node.children[j].record).GetFname()
		})

		for _, childNode := range node.children {

			msg := fmt.Sprintf(" %s %d", (*childNode.record).GetFname(), (*childNode.record).GetID())

			fmt.Print(msg)
			msgB.WriteString(msg)

		}

		logger.FSLogger.Info(msgB.String())
	}

}
