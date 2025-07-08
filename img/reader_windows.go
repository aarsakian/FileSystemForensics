package img

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"unsafe"

	"github.com/aarsakian/FileSystemForensics/logger"
	"golang.org/x/sys/windows"
)

var (
	kernel32             = windows.NewLazySystemDLL("kernel32.dll")
	procSetFilePointerEx = kernel32.NewProc("SetFilePointerEx")
)

type DISK_GEOMETRY struct {
	Cylinders         int64
	MediaType         int32
	TracksPerCylinder int32
	SectorsPerTrack   int32
	BytesPerSector    int32
}

type WindowsReader struct {
	a_file string
	fd     windows.Handle
}

func (winreader *WindowsReader) CreateHandler() {
	file_ptr, _ := windows.UTF16PtrFromString(winreader.a_file)
	var templateHandle windows.Handle
	fd, err := windows.CreateFile(file_ptr, windows.FILE_READ_DATA,
		windows.FILE_SHARE_READ, nil,
		windows.OPEN_EXISTING, 0, templateHandle)
	if err != nil {
		log.Fatalln(err)
	}
	winreader.fd = fd
}

func (winreader WindowsReader) CloseHandler() {
	windows.Close(winreader.fd)
}

func (winreader WindowsReader) GetDiskSize() int64 {
	const IOCTL_DISK_GET_DRIVE_GEOMETRY = 0x70000
	const nByte_DISK_GEOMETRY = 24
	disk_geometry := DISK_GEOMETRY{}

	var junk *uint32
	var inBuffer *byte
	err := windows.DeviceIoControl(winreader.fd, IOCTL_DISK_GET_DRIVE_GEOMETRY,
		inBuffer, 0, (*byte)(unsafe.Pointer(&disk_geometry)), nByte_DISK_GEOMETRY, junk, nil)
	if err != nil {
		log.Fatalln(err)
	}

	return disk_geometry.Cylinders * int64(disk_geometry.TracksPerCylinder) *
		int64(disk_geometry.SectorsPerTrack) * int64(disk_geometry.BytesPerSector)
}

func (winreader WindowsReader) ReadFile(startOffset int64, totalSize int) []byte {
	var wholebuffer bytes.Buffer
	w := bufio.NewWriter(&wholebuffer)

	const chunkSize = 512 * 1024 * 1024 // 512 MB

	buffer := make([]byte, chunkSize)
	bytesRead := int64(0)
	offset := int64(0)

	for bytesRead < int64(totalSize) {

		err := setFilePointerEx(winreader.fd, offset+startOffset, windows.FILE_BEGIN)

		if err != nil {
			panic(fmt.Sprintf("Seek failed at offset %d: %v", offset, err))
		}

		toRead := chunkSize
		if int64(totalSize)-offset < int64(chunkSize) {
			toRead = int(int64(totalSize) - offset)
		}

		var bytesRead uint32

		err = windows.ReadFile(winreader.fd, buffer[:toRead], &bytesRead, nil)
		if err != nil {
			panic(fmt.Sprintf("Read failed at offset %d: %v", offset, err))
		}

		w.Write(buffer)

		logger.FSLogger.Info(fmt.Sprintf("Read %d bytes at offset %d\n", bytesRead, offset))
		offset += int64(bytesRead)

		if bytesRead == 0 {
			break
		}
	}
	w.Flush()
	return wholebuffer.Bytes()[:totalSize]
}

func setFilePointerEx(handle windows.Handle, distance int64, moveMethod uint32) error {
	var newPos int64
	r1, _, err := procSetFilePointerEx.Call(
		uintptr(handle),
		uintptr(distance),
		uintptr(unsafe.Pointer(&newPos)),
		uintptr(moveMethod),
	)
	if r1 == 0 {
		return err
	}
	return nil
}
