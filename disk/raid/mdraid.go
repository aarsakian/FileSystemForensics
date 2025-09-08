package raid

type Superblock struct {
	Magic             [4]byte //4bytes 0xa92b4efc (Hexify(Bytereverse
	MajorVersion      uint32
	FeatureMap        uint32
	Pad0              [4]byte //zeros
	UUID              [16]byte
	RaidName          [32]byte //32bytes
	CTime             uint64   //C time
	RaidLevel         uint32
	RaidLayout        uint32
	Size              uint64 //sectors
	ChunkSize         uint32
	DevicesNum        uint32 //-96
	Uknown            [32]byte
	DataOffset        uint64 //136
	DataSize          uint64
	SuperOffset       uint64
	Uknown2           [8]byte
	DevNum            uint32 //160-
	NofCorrectedReads uint32
	DeviceUUID        [16]byte
	DeviceFlags       uint8
	BBlogShift        uint8
	BBlogSize         uint16
	BBlogOffset       uint32
	UTime             uint64 // C time
	Events            uint64
	ReSyncOffset      uint64
	SBChksum          [4]byte
	MaxDevices        uint32
	Pad32             [32]byte
	DevRoles          []DevRole
}

// role in array
type DevRole struct {
	Role uint32
}
