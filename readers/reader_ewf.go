package readers

import (
	"path"
	"strings"

	ewfLib "github.com/aarsakian/EWF_Reader/ewf"
	ewfutils "github.com/aarsakian/EWF_Reader/ewf/utils"
)

type EWFReader struct {
	PathToEvidenceFiles string
	fd                  ewfLib.EWF_Image
}

func (imgreader *EWFReader) CreateHandler() {
	extension := path.Ext(imgreader.PathToEvidenceFiles)
	if strings.ToLower(extension) == ".e01" {
		var ewf_image ewfLib.EWF_Image
		filenames := ewfutils.FindEvidenceFiles(imgreader.PathToEvidenceFiles)

		ewf_image.ParseEvidence(filenames)

		imgreader.fd = ewf_image
	}

}

func (imgreader EWFReader) CloseHandler() {

}

func (imgreader EWFReader) ReadFile(physicalOffset int64, length int) []byte {
	return imgreader.fd.RetrieveData(physicalOffset, int64(length))
}

func (imgreader EWFReader) GetDiskSize() int64 {
	return int64(imgreader.fd.Chunksize) * int64(imgreader.fd.NofChunks)
}
