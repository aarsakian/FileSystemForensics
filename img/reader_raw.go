package img

import (
	"fmt"
	"os"

	"github.com/aarsakian/FileSystemForensics/logger"
)

type RawReader struct {
	PathToEvidenceFiles string
	fd                  *os.File
}

func (imgreader *RawReader) CreateHandler() {
	file, err := os.Open(imgreader.PathToEvidenceFiles)
	if err != nil {
		// handle the error here
		fmt.Printf("err %s in getting handle of file ", err)
		return
	}
	imgreader.fd = file
}

func (imgreader RawReader) CloseHandler() {
	imgreader.fd.Close()
}

func (imgreader RawReader) ReadFile(physicalOffset int64, length int) []byte {

	data := make([]byte, length)
	_, err := imgreader.fd.ReadAt(data, int64(physicalOffset))
	msg := fmt.Sprintf("raw read: offset %d len %d", physicalOffset, length)
	logger.MFTExtractorlogger.Info(msg)
	if err != nil {
		msg := fmt.Sprintf("error %s reading  file ", err)
		logger.MFTExtractorlogger.Error(msg)
		fmt.Printf("%s\n", msg)
		return nil
	}
	return data

}

func (imgreader RawReader) GetDiskSize() int64 {
	finfo, err := os.Stat(imgreader.PathToEvidenceFiles)
	if err != nil {
		return -1
	}
	return int64(finfo.Size())
}
