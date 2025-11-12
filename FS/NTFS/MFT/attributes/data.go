package attributes

import "fmt"

var dataFlags = map[uint16]string{
	0x0000: "None",
	0x0001: "Compressed",
	0x4000: "Encrypted",
	0x8000: "Encrypted",
}

type DATA struct {
	Content []byte
	Header  *AttributeHeader
}

func (data *DATA) SetHeader(header *AttributeHeader) {
	data.Header = header
}

func (data *DATA) Parse(datab []byte) {
	copy(data.Content, datab)
}

func (data DATA) GetHeader() AttributeHeader {
	return *data.Header
}

func (data DATA) FindType() string {
	return data.Header.GetType()
}
func (data DATA) IsNoNResident() bool {
	return data.Header.IsNoNResident()
}

func (data DATA) ShowInfo() {
	fmt.Printf("type %s %t %s \n", data.FindType(), data.IsNoNResident(), data.GetType())
}

func (data DATA) GetType() string {
	var dataType string
	for key, value := range dataFlags {
		if key&data.Header.Flags != 0 {
			dataType += " " + value
		}
	}
	return dataType
}
