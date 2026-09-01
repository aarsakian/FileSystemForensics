package attributes

type Security struct {
	Header      *AttributeHeader
	Revision    uint8
	Sbz1        uint8
	Control     uint16
	OffsetOwner uint32
	OffsetGroup uint32
	OffsetSacl  uint32
	OffsetDacl  uint32
}

func (securitydescriptor Security) FindType() string {
	return securitydescriptor.Header.GetType()
}

func (securitydescriptor *Security) SetHeader(header *AttributeHeader) {
	securitydescriptor.Header = header
}

func (securitydescriptor Security) GetHeader() AttributeHeader {
	return *securitydescriptor.Header
}

func (securitydescriptor *Security) Parse(data []byte) {

}

func (securitydescriptor Security) GetInfo() string {
	return ""
}
func (securitydescriptor Security) IsNoNResident() bool {
	return securitydescriptor.Header.IsNoNResident()
}
