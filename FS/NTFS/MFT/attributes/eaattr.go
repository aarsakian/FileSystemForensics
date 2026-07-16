package attributes

import "github.com/aarsakian/FileSystemForensics/utils"

type EA_INFORMATION struct {
	SizeOfEntry uint16
	NofEA       uint16
	Size        uint32
	Header      *AttributeHeader
}

type ExtendedAttribute struct {
	NextAttrOffset int32
	Flags          uint8
	NofChars       uint8
	DataSize       uint16
	Name           []byte
	Value          []byte
	Header         *AttributeHeader
}

func (ea_info EA_INFORMATION) FindType() string {
	return ea_info.Header.GetType()
}

func (ea_info *EA_INFORMATION) SetHeader(header *AttributeHeader) {
	ea_info.Header = header
}

func (ea_info EA_INFORMATION) GetHeader() AttributeHeader {
	return *ea_info.Header
}

func (ea_info EA_INFORMATION) IsNoNResident() bool {
	return ea_info.Header.IsNoNResident()
}

func (ea_info *EA_INFORMATION) Parse(data []byte) {

	utils.Unmarshal(data, ea_info)

}

func (ea_info *EA_INFORMATION) GetInfo() string {
	return ""
}

func (ea ExtendedAttribute) FindType() string {
	return ea.Header.GetType()
}

func (ea *ExtendedAttribute) SetHeader(header *AttributeHeader) {
	ea.Header = header
}

func (ea ExtendedAttribute) GetHeader() AttributeHeader {
	return *ea.Header
}

func (ea ExtendedAttribute) IsNoNResident() bool {
	return ea.Header.IsNoNResident()
}

func (ea *ExtendedAttribute) Parse(data []byte) {

	utils.Unmarshal(data, ea)

}

func (ea ExtendedAttribute) GetInfo() string {
	return utils.RemoveNulls(ea.Name)
}
