package datums

import (
	"fmt"
	"strings"

	"github.com/aarsakian/FileSystemForensics/utils"
)

// FVEVolumeMasterKey represents the VMK entry.
// Value type: 0x0008
type FVEVolumeMasterKey struct {
	Header               *DatumHeader      // Common header for all entries
	KeyIdentifier        [16]byte          // GUID identifying this VMK
	LastModificationTime utils.WindowsTime // FILETIME of last modification
	Unknown              uint16            // Unknown field
	ProtectionType       uint16            // Protection type (see VMKProtectionType constants)
	Datums               []Datum           // Variable array of additional datums (property entries, validation data, etc.)
}

func (vmk *FVEVolumeMasterKey) Process(raw []byte) error {
	if _, err := utils.Unmarshal(raw, vmk); err != nil {
		return err
	}
	offset := 0

	vmkDatumSize := 28

	for offset < len(raw[vmkDatumSize:]) {
		datumHeader := new(DatumHeader)
		err := datumHeader.Process(raw[vmkDatumSize+offset:])
		if err != nil {
			break
		}
		if datumHeader.EntrySize == 0 {
			break
		}

		if offset+int(datumHeader.EntrySize) > len(raw) {
			break
		}

		datum, err := CreateDatum(*datumHeader, raw[vmkDatumSize+offset+8:vmkDatumSize+offset+int(datumHeader.EntrySize)])
		if err != nil {
			break
		}

		vmk.Datums = append(vmk.Datums, datum)
		offset += int(datumHeader.EntrySize)
	}

	return nil
}

func (vmk *FVEVolumeMasterKey) SetHeader(header *DatumHeader) {
	vmk.Header = header
}

func (vmk *FVEVolumeMasterKey) GetHeader() *DatumHeader {
	return vmk.Header
}

func (vmk *FVEVolumeMasterKey) GetInfo() string {
	var nestedInfo strings.Builder
	for _, datum := range vmk.Datums {
		nestedInfo.WriteString(fmt.Sprintf("\n  - %s", datum.GetInfo()))
	}
	return fmt.Sprintf("Header info %s FVEVolumeMasterKey: Properties Length: %d bytes Protection Type %s Time %s Nested datums %s",
		vmk.Header.GetInfo(), len(vmk.Datums), vmk.GetProtectionType(),
		vmk.LastModificationTime.ConvertToIsoTime(), nestedInfo.String())

}

/*func (vmk FVEVolumeMasterKey) Get_Key() FVEKey {

}*/
