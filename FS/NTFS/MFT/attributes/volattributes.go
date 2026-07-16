package attributes

import "github.com/aarsakian/FileSystemForensics/utils"

type VolumeName struct {
	Name   string
	Header *AttributeHeader
}

type VolumeInfo struct {
	F1     uint64 //unused
	MajVer uint8  // 8-8
	MinVer uint8  // 9-9
	Flags  uint16 //see table 13.22
	F2     uint32
	Header *AttributeHeader
}

func (volInfo *VolumeInfo) SetHeader(header *AttributeHeader) {
	volInfo.Header = header
}

func (volInfo VolumeInfo) GetHeader() AttributeHeader {
	return *volInfo.Header
}

func (volInfo *VolumeInfo) Parse(data []byte) {
	utils.Unmarshal(data, volInfo)
}

func (volInfo VolumeInfo) IsNoNResident() bool {
	return volInfo.Header.IsNoNResident()
}

func (volInfo VolumeInfo) FindType() string {
	return volInfo.Header.GetType()
}

func (volinfo VolumeInfo) GetInfo() string {
	return ""
}

func (volName *VolumeName) SetHeader(header *AttributeHeader) {
	volName.Header = header
}

func (volName VolumeName) GetHeader() AttributeHeader {
	return *volName.Header
}

func (volName *VolumeName) Parse(data []byte) {
	volName.Name = utils.RemoveNulls(data)

}

func (volName VolumeName) FindType() string {
	return volName.Header.GetType()
}

func (volName VolumeName) IsNoNResident() bool {
	return volName.Header.IsNoNResident()
}

func (volName VolumeName) GetInfo() string {
	return volName.Name
}
