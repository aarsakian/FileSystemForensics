package VSS

import (
	"fmt"

	"github.com/aarsakian/FileSystemForensics/readers"
	"github.com/aarsakian/FileSystemForensics/utils"
)

type ShadowVolume struct {
	Header   *Header
	Catalogs []Catalog
	Stores   []Store
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
	VssGUID       [16]byte
	Version       uint32
	RecordType    uint32
	ShadowOffset  int64
	CurrentOffset int64 //from start of volume block catalog
	NextOffset    int64 //next catalog block
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

	data, _ := handler.ReadFile(partitionOffsetB+7680, 512)

	shadowVol.Header = new(Header)
	utils.Unmarshal(data, shadowVol.Header)

	data, _ = handler.ReadFile(partitionOffsetB+shadowVol.Header.CatalogOffset, 16384)

	for {

		catalogHeader := new(CatalogHeader)
		utils.Unmarshal(data[:128], catalogHeader)
		offset := 128
	loop:
		for offset < len(data) {
			switch data[offset] {
			case 0x01:
				entry1 := new(CatalogEntry1)
				readBytes, _ := utils.Unmarshal(data[offset:], entry1)
				offset += readBytes

				entries1 = append(entries1, *entry1)
			case 0x02:
				entry2 := new(CatalogEntry2)
				readBytes, _ := utils.Unmarshal(data[offset:], entry2)
				offset += readBytes

				entries2 = append(entries2, *entry2)
			case 0x03:
				entry3 := new(CatalogEntry3)
				readBytes, _ := utils.Unmarshal(data[offset:], entry3)
				offset += readBytes

				entries3 = append(entries3, *entry3)
			default:
				break loop
			}

		}
		if catalogHeader.NextOffset == 0 {
			break
		}
		data, _ = handler.ReadFile(partitionOffsetB+catalogHeader.NextOffset, 16384)
		shadowVol.Catalogs = append(shadowVol.Catalogs, Catalog{Header: catalogHeader, EntriesType1: entries1,
			EntriesType2: entries2, EntriesType3: entries3})

	}

	shadowVol.ProcessStores(handler, partitionOffsetB)

}

func (shadowVol *ShadowVolume) ProcessStores(handler readers.DiskReader, partitionOffsetB int64) {
	var stores []Store
	for _, catalog := range shadowVol.Catalogs {
		for _, entry3 := range catalog.EntriesType3 {
			stor := new(Store)

			data, _ := handler.ReadFile(partitionOffsetB+int64(entry3.StoreHeaderOffset), 16384)

			stor.Process(data)

			storeList := new(StoreList)
			storeList.Header = new(StoreHeader)

			data, _ = handler.ReadFile(partitionOffsetB+int64(entry3.StoreBlockListOffset), 16384)
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

			data, _ = handler.ReadFile(partitionOffsetB+int64(entry3.StoreBlockRangeListOffset), 16384)
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
			data, _ = handler.ReadFile(partitionOffsetB+int64(entry3.StoreBitmapOffset), 16384)
			utils.Unmarshal(data, storeBitmapHeader)

			stor.BitmapData = make([]byte, len(data[128:]))
			copy(stor.BitmapData, data[128:])

			storePrevBitmapHeader := new(StoreHeader)
			data, _ = handler.ReadFile(partitionOffsetB+int64(entry3.StorePrevBitmapOffset), 16384)
			utils.Unmarshal(data, storePrevBitmapHeader)

			stor.PrevBitmapData = make([]byte, len(data[128:]))
			copy(stor.PrevBitmapData, data[128:])

			stores = append(stores, *stor)
		}
	}
	shadowVol.Stores = stores

}

func (shadow ShadowVolume) GetClustersInfo(clusters []int) []int {
	shadowClusterOffsets := make([]int, 2)
	for idx, cluster := range clusters {
		shadowClusterOffsets[idx] = shadow.GetShadowClusterOffset([]int{cluster})

	}
	return shadowClusterOffsets
}

func (shadow ShadowVolume) GetShadowClusterOffset(clusters []int) int {
	for _, store := range shadow.Stores {
		for _, blockrangeList := range store.StoreBlockRange.BlockRangeLists {
			if blockrangeList.HasClusters(clusters[0]) {
				return int(blockrangeList.ShadowOffset)/4096 + clusters[0] - int(blockrangeList.StartOffset/4096)
			}
		}

		for _, blockDescriptor := range store.StoresList.BlockDescriptors {
			if int(blockDescriptor.OriginalDataBlockOffset) == clusters[0]*4096 {
				return int(blockDescriptor.ShadowDataBlockOffset)
			}
		}
	}
	return -1

}

func (shadow ShadowVolume) ListVSS() {
	for _, store := range shadow.Stores {
		fmt.Printf("%s \n", store.Header.GetInfo())
		if store.Info != nil {
			fmt.Printf(" %s Machine ID %s Service ID %s\n", store.Info.DecodeStoreAttributeFlags(),
				store.Info.OperatingMachineName, store.Info.ServiceMachineName)
		}
		fmt.Printf("Block Ranges %s\n", store.StoreBlockRange.Header.GetInfo())

		for _, blockrangeList := range store.StoreBlockRange.BlockRangeLists {
			fmt.Printf("%d -> %d\n",
				blockrangeList.StartOffset/4096,
				blockrangeList.StartOffset/4096+blockrangeList.RangeSize/4096)
		}
		fmt.Printf("Store Blocks: %s\n", store.StoresList.Header.GetInfo())
		for _, blockDescriptor := range store.StoresList.BlockDescriptors {
			if blockDescriptor.OriginalDataBlockOffset == 0 {
				continue
			}
			fmt.Printf("%s : %d -> %d \t",
				blockDescriptor.DecodeStoreBlockFlags(),
				blockDescriptor.OriginalDataBlockOffset/4096,
				blockDescriptor.StoreDataBlockOffset/4096)
		}
		fmt.Printf("\n")
	}

}
