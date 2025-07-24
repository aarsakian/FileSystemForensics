package attributes

var ItemTypes = map[int]string{
	0x01: "INODE_ITEM",
	0x0C: "INODE_REF",
	0x0D: "INODE_EXTREF",
	0xB0: "TREE_BLOCK_REF",
	0x18: "XATTR_ITEM",
	0x30: "ORPHAN_ITEM",
	0x48: "DIR_LOG_INDEX",
	0x54: "DIR_ITEM",
	0x60: "DIR_INDEX",
	0x6C: "EXTENT_DATA",
	0x80: "EXTENT_CSUM",
	0x84: "ROOT_ITEM",
	0x90: "ROOT_BACKREF",
	0x9C: "ROOT_REF",
	0xA8: "EXTENT_ITEM",
	0xA9: "METADATA_ITEM",
	0xB2: "EXTENT_DATA_REF",
	0xB4: "EXTENT_REF_V0",
	0xB6: "SHARED_BLOCK_REF",
	0xB8: "SHARED_DATA_REF",
	0xC0: "BLOCK_GROUP_ITEM",
	0xC6: "FREE_SPACE_INFO",
	0xC7: "FREE_SPACE_EXTENT",
	0xC8: "FREE_SPACE_BITMAP",
	0xC9: "PERSISTENT_ITEM",
	0xCC: "DEV_EXTENT",
	0x3C: "DIR_LOG_ITEM",
	0xD8: "DEV_ITEM",
	0xE4: "CHUNK_ITEM",
	0xF8: "TEMPORARY_ITEM",
	0xF9: "DEV_STATS",
	0xFD: "STRING_ITEM",
}

var DirTypes = map[uint8]string{
	0: "BTRFS_TYPE_UNKNOWN",
	1: "BTRFS_TYPE_FILE",
	2: "BTRFS_TYPE_DIRECTORY",
	3: "BTRFS_TYPE_CHARDEV",
	4: "BTRFS_TYPE_BLOCKDEV",
	5: "BTRFS_TYPE_FIFO",
	6: "BTRFS_TYPE_SOCKET",
	7: "BTRFS_TYPE_SYMLINK",
	8: "BTRFS_TYPE_EA",
}

var ExtentTypes = map[uint8]string{
	0: "Inline Extent",
	1: "Regular Extent",
	2: "Pre-alloc Extent",
}

var ExtentItemTypes = map[uint8]string{
	1:   "BTRFS_EXTENT_FLAG_DATA",
	2:   "BTRFS_EXTENT_FLAG_TREE_BLOCK",
	128: "BTRFS_BLOCK_FLAG_FULL_BACKREF",
}

var RefTypes = map[uint8]string{
	0: "ROOT_REF",
	1: "ROOT_BACKREF",
}

var InodeTypes = map[uint64]string{
	1:    "NODATASUM",
	2:    "NODATACOW",
	4:    "READONLY",
	8:    "NOCOMPRESS",
	16:   "PREALLOC",
	32:   "SYNC",
	64:   "IMMUTABLE",
	128:  "APPEND",
	256:  "NODUMP",
	512:  "NOATIME",
	1024: "DIRSYNC",
	2048: "COMPRESS",
}

var Block_Group_Types = map[int]string{
	1:   "BLOCK_GROUP_DATA",
	8:   "BLOCK_GROUP_RAID0",
	16:  "BLOCK_GROUP_RAID1",
	32:  "BLOCK_GROUP_DUPLIATE",
	64:  "BLOCK_GROUP_RAID10",
	128: "BLOCK_GROUP_RAID5",
	256: "BLOCK_GROUP_RAID6",
}

var ObjectTypes = map[int]string{
	0:  "DEV_STATS",
	1:  "ROOT_TREE",
	2:  "EXTENT_TREE",
	3:  "CHUNK_TREE",
	4:  "DEV_TREE",
	5:  "FS_TREE",
	6:  "ROOT_DIR_TREE",
	7:  "CSUM_TREE",
	8:  "QUOTA_TREE",
	9:  "UUID_TREE",
	10: "FREE_SPACE",
	11: "BLOCK_GROUP",
	12: "RAID_STRIPE",
}

var CompressionTypes = map[int]string{
	0: "No compression", 1: "Zlib", 2: "LZO", 3: "ZSTD",
}
