package readers

type DiskReader interface {
	CreateHandler()
	CloseHandler()
	ReadFile(int64, int) ([]byte, error)
	GetDiskSize() int64
}

func GetHandler(pathToDisk string, mode string) DiskReader {

	var dr DiskReader
	switch mode {
	case "physicalDrive":
		dr = &WindowsReader{a_file: pathToDisk}
	case "linux":
		//	dr = UnixReader{pathToDisk: pathToDisk}
	case "ewf":
		dr = &EWFReader{PathToEvidenceFiles: pathToDisk}
	case "raw":
		dr = &RawReader{PathToEvidenceFiles: pathToDisk}
	case "vmdk":
		dr = &VMDKReader{PathToEvidenceFiles: pathToDisk}
	}
	dr.CreateHandler()

	return dr
}
