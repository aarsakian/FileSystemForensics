package VSS

import (
	"fmt"

	"github.com/aarsakian/FileSystemForensics/readers"
	"github.com/aarsakian/FileSystemForensics/utils"
)

type ShadowVolume struct {
	Catalog *Catalog
	Stores  []Store
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

func (shadowVol *ShadowVolume) Process(handler readers.DiskReader, partitionOffsetB int64) {
	var entries1 []CatalogEntry1
	var entries2 []CatalogEntry2
	var entries3 []CatalogEntry3

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

	shadowVol.Catalog = &Catalog{Header: catalogHeader, EntriesType1: entries1,
		EntriesType2: entries2, EntriesType3: entries3}
	shadowVol.ProcessStores(handler, partitionOffsetB)

}

func (shadowVol *ShadowVolume) ProcessStores(handler readers.DiskReader, partitionOffsetB int64) {
	var stores []Store
	for _, entry3 := range shadowVol.Catalog.EntriesType3 {
		stor := new(Store)

		data := handler.ReadFile(partitionOffsetB+int64(entry3.StoreHeaderOffset), 16384)

		stor.Process(data)

		storeList := new(StoreList)
		storeList.Header = new(StoreHeader)

		data = handler.ReadFile(partitionOffsetB+int64(entry3.StoreBlockListOffset), 16384)
		readBytes, _ := utils.Unmarshal(data, storeList.Header)
		offset := readBytes

		for offset < len(data) {
			storeBlockDescriptor := new(StoreBlockDescriptor)

			readBytes, _ := utils.Unmarshal(data[offset:], storeBlockDescriptor)
			offset += readBytes
			storeList.BlockDescriptors = append(storeList.BlockDescriptors, *storeBlockDescriptor)
		}

		stor.StoresList = storeList

		storeBlockRange := new(StoreBlockRange)
		storeBlockRange.Header = new(StoreHeader)

		data = handler.ReadFile(partitionOffsetB+int64(entry3.StoreBlockRangeListOffset), 16384)
		readBytes, _ = utils.Unmarshal(data, storeBlockRange.Header)
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
			storeBlockRange.BlockRangeLists = append(storeBlockRange.BlockRangeLists, *storeRangeEntry)
		}

		stor.StoreBlockRange = storeBlockRange

		storeBitmapHeader := new(StoreHeader)
		data = handler.ReadFile(partitionOffsetB+int64(entry3.StoreBitmapOffset), 16384)
		utils.Unmarshal(data, storeBitmapHeader)

		stor.BitmapData = make([]byte, len(data[128:]))
		copy(stor.BitmapData, data[128:])

		storePrevBitmapHeader := new(StoreHeader)
		data = handler.ReadFile(partitionOffsetB+int64(entry3.StorePrevBitmapOffset), 16384)
		utils.Unmarshal(data, storePrevBitmapHeader)

		stor.PrevBitmapData = make([]byte, len(data[128:]))
		copy(stor.PrevBitmapData, data[128:])

		stores = append(stores, *stor)
	}
	shadowVol.Stores = stores

}
