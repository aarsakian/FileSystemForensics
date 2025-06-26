package attributes

import "github.com/aarsakian/FileSystemForensics/utils"

// c BTRFS_DEV_ITEM_KEYs which help map physical offsets to logical offsets
type DevItem struct { //98B
	DeviceID           uint64
	NofBytes           uint64
	NofUsedBytes       uint64
	OptimalIOAlignment uint32
	OptimalIOWidth     uint32
	MinimailIOSize     uint32
	Type               uint64
	Generation         uint64
	StartOffset        uint64
	DevGroup           uint32
	SeekSpeed          uint8
	Bandwidth          uint8
	DeviceUUID         [16]byte
	FilesystemUUID     [16]byte
}

func (devItem DevItem) ShowInfo() {

}

func (devItem DevItem) GetInfo() string {
	return ""
}

func (devItem *DevItem) Parse(data []byte) int {
	offset, _ := utils.Unmarshal(data, devItem)
	return offset
}
