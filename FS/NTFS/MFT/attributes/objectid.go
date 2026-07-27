package attributes

import (
	"fmt"
	"strings"

	"github.com/aarsakian/FileSystemForensics/utils"
)

type ObjectID struct { //unique guID
	ObjGUID     [16]byte //object ID
	OrigVolGUID [16]byte //volume ID
	OrigObjGUID [16]byte //original objID
	OrigDomGUID [16]byte // domain ID
	Header      *AttributeHeader
}

func (objectId *ObjectID) SetHeader(header *AttributeHeader) {
	objectId.Header = header
}

func (objectId ObjectID) GetHeader() AttributeHeader {
	return *objectId.Header
}

func (objectId *ObjectID) Parse(data []byte) {
	utils.Unmarshal(data, objectId)
}

func (objectId ObjectID) FindType() string {
	return objectId.Header.GetType()
}

func (objectId ObjectID) IsNoNResident() bool {
	return objectId.Header.IsNoNResident()
}

func (objectId ObjectID) GetInfo() string {
	var txt strings.Builder

	fmt.Fprintf(&txt, "%s", objectId.Header.GetInfo())

	fmt.Fprintf(&txt, "type %s\n", objectId.FindType())
	return txt.String()
}
