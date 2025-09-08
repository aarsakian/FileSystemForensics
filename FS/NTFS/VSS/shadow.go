package vss

import (
	"fmt"

	"github.com/aarsakian/FileSystemForensics/readers"
	"github.com/aarsakian/FileSystemForensics/utils"
)

type VSS struct {
	Catalog *Catalog
}

type Header struct {
	GUID                      [16]byte
	Version                   uint32
	RecordType                uint32
	CurrentOffset             uint64
	Uknown                    [16]byte
	CatalogOffset             int64
	MaxSize                   uint64
	VolumeID                  [16]byte
	ShadowCopyStorageVolumeID [16]byte
}

type CatalogHeader struct {
	VssGUID        [16]byte
	Version        uint32
	RecordType     uint32
	RelativeOffset int64
	CurrentOffset  int64
	NextOffset     int64
}

type Catalog struct {
	Header       *CatalogHeader
	EntriesType1 []CatalogEntry1
	EntriesType2 []CatalogEntry2
	EntriesType3 []CatalogEntry3
}

type CatalogEntry1 struct {
	EntryType uint64
	Uknown    [120]byte
}

type CatalogEntry2 struct {
	EntryType    uint64
	VolumeSize   uint64
	StoreGUID    [16]byte
	Uknown       [16]byte
	CreationTime utils.WindowsTime
	Uknown2      [72]byte
}

type CatalogEntry3 struct {
	EntryType                 uint64
	StoreBlockListOffset      uint64
	StoreGUID                 [16]byte
	StoreHeaderOffset         uint64
	StoreBlockRangeListOffset uint64
	StoreBitmapOffset         uint64
	ParRef                    uint64
	ParSeq                    uint16
	AllocatedSize             uint64
	StorePrevBitmapOffset     uint64
	Uknown                    [48]byte
}

func ProcessVSS(handler readers.DiskReader, partitionOffsetB int64) {
	var entries1 []CatalogEntry1
	var entries2 []CatalogEntry2
	var entries3 []CatalogEntry3
	_vss := VSS{}
	data := handler.ReadFile(partitionOffsetB+7680, 512)
	header := new(Header)
	utils.Unmarshal(data, header)

	data = handler.ReadFile(partitionOffsetB+header.CatalogOffset, 16384)
	catalogHeader := new(CatalogHeader)
	utils.Unmarshal(data[:128], catalogHeader)
	offset := 128

	for offset < len(data) {
		if data[offset] == 0x01 {
			entry1 := new(CatalogEntry1)
			readBytes, _ := utils.Unmarshal(data[offset:], entry1)
			offset += readBytes

			entries1 = append(entries1, *entry1)
		} else if data[offset] == 0x02 {
			entry2 := new(CatalogEntry2)
			readBytes, _ := utils.Unmarshal(data[offset:], entry2)
			offset += readBytes

			entries2 = append(entries2, *entry2)
		} else if data[offset] == 0x03 {
			entry3 := new(CatalogEntry3)
			readBytes, _ := utils.Unmarshal(data[offset:], entry3)
			offset += readBytes

			entries3 = append(entries3, *entry3)
		} else {
			break
		}

	}

	cat := Catalog{Header: catalogHeader, EntriesType1: entries1,
		EntriesType2: entries2, EntriesType3: entries3}
	_vss.Catalog = &cat
	for _, entry3 := range cat.EntriesType3 {
		stor := new(Store)

		storeHeader := new(StoreHeader)
		data = handler.ReadFile(partitionOffsetB+int64(entry3.StoreHeaderOffset), 16384)
		readBytes, _ := utils.Unmarshal(data, storeHeader)
		offset = readBytes

		storeInfo := new(StoreInfo)
		readBytes, _ = utils.Unmarshal(data[offset:], storeInfo)
		offset += readBytes - 2 //workaround it counts ServiceMachineNameSize

		storeInfo.OperatingMachineName = utils.DecodeUTF16(data[offset : offset+int(storeInfo.OperatingMachineNameSize)])
		offset += int(storeInfo.OperatingMachineNameSize)

		storeInfo.ServiceMachineNameSize = utils.ToUint16(data[offset:])
		offset += 2
		storeInfo.ServiceMachineName = utils.DecodeUTF16(data[offset : offset+int(storeInfo.ServiceMachineNameSize)])

		storeListHeader := new(StoreHeader)
		data = handler.ReadFile(partitionOffsetB+int64(entry3.StoreBlockListOffset), 16384)
		readBytes, _ = utils.Unmarshal(data, storeListHeader)
		offset = readBytes

		for offset < len(data) {
			storeList := new(StoreList)

			readBytes, _ := utils.Unmarshal(data[offset:], storeList)
			offset += readBytes
			stor.BlockLists = append(stor.BlockLists, *storeList)
		}

		storeRangeHeader := new(StoreHeader)
		data = handler.ReadFile(partitionOffsetB+int64(entry3.StoreBlockRangeListOffset), 16384)
		readBytes, _ = utils.Unmarshal(data, storeRangeHeader)
		offset = readBytes

		for offset < len(data) {
			storeRangeEntry := new(StoreBlockRangeEntry)
			readBytes, err := utils.Unmarshal(data[offset:], storeRangeEntry)
			if storeRangeEntry.StartOffset == 0 {
				break
			}
			if err != nil {
				fmt.Println(err)
				break
			}
			offset += readBytes
			stor.BlockRangeLists = append(stor.BlockRangeLists, *storeRangeEntry)
		}

		storeBitmapHeader := new(StoreHeader)
		data = handler.ReadFile(partitionOffsetB+int64(entry3.StoreBitmapOffset), 16384)
		utils.Unmarshal(data, storeBitmapHeader)

		storePrevBitmapHeader := new(StoreHeader)
		data = handler.ReadFile(partitionOffsetB+int64(entry3.StorePrevBitmapOffset), 16384)
		utils.Unmarshal(data, storePrevBitmapHeader)
	}

}
