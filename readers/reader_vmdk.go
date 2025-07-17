package img

import (
	"fmt"
	"path"
	"strings"

	"github.com/aarsakian/VMDK_Reader/logger"
	"github.com/aarsakian/VMDK_Reader/vmdk"
)

type VMDKReader struct {
	PathToEvidenceFiles string
	fd                  vmdk.VMDKImage //Handle
}

func (imgreader *VMDKReader) CreateHandler() {
	extension := path.Ext(imgreader.PathToEvidenceFiles)
	if strings.ToLower(extension) == ".vmdk" {

		vmdkimage := vmdk.VMDKImage{Path: imgreader.PathToEvidenceFiles}
		vmdkimage.Process()

		if vmdkimage.HasParent() {
			parentVMDKImage, err := vmdkimage.LocateParent()
			if err != nil {
				logger.VMDKlogger.Error(err)
			} else {
				parentVMDKImage.Process()
				vmdkimage.ParentImage = &parentVMDKImage
			}
		}
		imgreader.fd = vmdkimage

	} else {
		panic("only VMDK Sparse  images are supported")
	}

}

func (imgreader VMDKReader) CloseHandler() {

}

func (imgreader VMDKReader) ReadFile(physicalOffset int64, length int) []byte {
	logger.VMDKlogger.Info(fmt.Sprintf("Read from %d len %d", physicalOffset, length))
	return imgreader.fd.RetrieveData(physicalOffset, int64(length))
}

func (imgreader VMDKReader) GetDiskSize() int64 {
	return imgreader.fd.GetHDSize()
}
