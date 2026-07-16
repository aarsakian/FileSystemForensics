package attributes

import "github.com/aarsakian/FileSystemForensics/utils"

type LoggedUtilityStream struct {
	Kind    string
	TXFDATA *TXFDATA
	Header  *AttributeHeader
}

type TXFDATA struct {
	ParRef              uint64
	ParSeq              uint16
	Flags               [8]byte //
	USN                 uint64  //
	TxId                uint64  //
	LSN_NTFS_Metadata   uint64  //
	LSN_User_Data       uint64  //
	LSN_Directory_Index uint64  //

}

func (loggedUtility LoggedUtilityStream) FindType() string {
	return loggedUtility.Header.GetType()
}

func (loggedUtility *LoggedUtilityStream) SetHeader(header *AttributeHeader) {
	loggedUtility.Header = header
}

func (loggedUtility LoggedUtilityStream) GetHeader() AttributeHeader {
	return *loggedUtility.Header
}

func (loggedUtility LoggedUtilityStream) IsNoNResident() bool {
	return loggedUtility.Header.IsNoNResident()
}

func (loggedUtility *LoggedUtilityStream) Parse(data []byte) {
	if loggedUtility.Kind == "$TXF_DATA" {
		txfData := new(TXFDATA)
		utils.Unmarshal(data, txfData)
		loggedUtility.TXFDATA = txfData
	}

}

func (loggedUtiltiy LoggedUtilityStream) GetInfo() string {
	return ""
}
