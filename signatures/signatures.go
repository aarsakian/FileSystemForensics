package signatures

import (
	"embed"
	"encoding/csv"
	"encoding/hex"
	"log"
	"strconv"
	"strings"
)

//go:embed signatures.csv
var signaturesFS embed.FS

type SignatureNode struct {
	IsTerminal  bool   // True if a signature ends exactly at this byte depth
	FormatName  string // The matched extension/MIME
	Occurrences int    // Localized heuristic weight
	Children    map[byte]*SignatureNode
}

type SignatureManager struct {
	Root *SignatureNode
}

// FindSignature positionally tracks depth perfectly with the byte index
func (sgm SignatureManager) FindSignature(fileHeader []byte) string {
	nextNode := sgm.Root

	// i directly represents the tree level/depth we are looking for
	for i := 0; i < len(fileHeader); i++ {
		targetByte := fileHeader[i]

		for nextNode != nil {
			child, ok := nextNode.Children[targetByte]
			if ok {
				nextNode = child

			} else {
				break
			}

			// If this specific length yields a valid file format, check it
			if nextNode.IsTerminal {
				return nextNode.FormatName
			}

		}

	}

	return "Unknown Format"
}

func (sgm *SignatureManager) BuildSignatureTree(signatures map[string][]byte) {
	//var prevNode *SignatureNode
	sgm.Root = &SignatureNode{Children: map[byte]*SignatureNode{}}
	for formatName, signature := range signatures {
		sgm.Root.Insert(signature, formatName)
	}

}

func (node *SignatureNode) Insert(signature []byte, formatName string) {

	for _, sigval := range signature {
		childNode, ok := node.Children[sigval]
		if ok {
			childNode.Occurrences++
		} else {
			childNode = &SignatureNode{Occurrences: 1, Children: map[byte]*SignatureNode{}, IsTerminal: true}
		}

		node.Children[sigval] = childNode
		node.IsTerminal = false

		node = childNode
	}
	node.FormatName = formatName

}

func ReadSignatures() map[string][]byte {
	signatures := make(map[string][]byte)
	f, err := signaturesFS.Open("signatures.csv")
	if err != nil {
		log.Fatal(err)
	}
	r := csv.NewReader(f)

	signaturesContent, err := r.ReadAll()
	if err != nil {
		log.Fatal(err)
	}

	for rowid, row := range signaturesContent {
		if rowid == 0 {
			continue
		}
		//expand the singature in case offset (row[1])>0
		len, _ := strconv.Atoi(row[1])
		signature := make([]byte, len)

		b, _ := hex.DecodeString(strings.ReplaceAll(row[0], " ", ""))
		signature = append(signature, b...)

		signatures[row[2]] = signature
	}
	return signatures

}
