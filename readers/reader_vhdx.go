package readers

import (
	"log"

	"github.com/aarsakian/VHD_Reader/vhdx"
)

type VHDXReader struct {
	PathToEvidenceFiles string
	fd                  *vhdx.Image
}

func (vhdxreader *VHDXReader) CreateHandler() {
	vhdxImage := new(vhdx.Image)
	err := vhdxImage.ParseEvidence(vhdxreader.PathToEvidenceFiles)
	if err != nil {
		log.Fatalf("Failed to parse evidence: %v", err)
	}
	vhdxreader.fd = vhdxImage

}

func (vhdxreader VHDXReader) CloseHandler() {
}

func (vhdxreader VHDXReader) GetDiskSize() int64 {
	return int64(vhdxreader.fd.VirtualSize)
}

func (vhdxreader VHDXReader) ReadFile(physicalOffset int64, length int) ([]byte, error) {
	data, err := vhdxreader.fd.RetrieveData(physicalOffset, int64(length))
	if err != nil {
		return nil, err
	}
	return data, nil
}
