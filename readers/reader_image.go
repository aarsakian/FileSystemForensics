package img

import (
	"path"
	"strings"

	ewfLib "github.com/aarsakian/EWF_Reader/ewf"

	"github.com/aarsakian/FileSystemForensics/utils"
)

type ImageReader struct {
	PathToEvidenceFiles string
	fd                  ewfLib.EWF_Image
}

func (imgreader *ImageReader) CreateHandler() {
	extension := path.Ext(imgreader.PathToEvidenceFiles)
	if strings.ToLower(extension) == ".e01" {
		var ewf_image ewfLib.EWF_Image
		filenames := utils.FindEvidenceFiles(imgreader.PathToEvidenceFiles)

		ewf_image.ParseEvidence(filenames)

		imgreader.fd = ewf_image
	} else {
		panic("only EWF  images are supported")
	}

}

func (imgreader ImageReader) CloseHandler() {

}

func (imgreader ImageReader) ReadFile(physicalOffset int64, length int) []byte {
	return imgreader.fd.RetrieveData(physicalOffset, int64(length))
}

func (imgreader ImageReader) GetDiskSize() int64 {
	return int64(imgreader.fd.Chuncksize) * int64(imgreader.fd.NofChunks)
}
