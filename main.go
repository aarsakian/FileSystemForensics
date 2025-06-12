package main

import (
	//"C"

	//"database/sql"

	"flag"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	EWFLogger "github.com/aarsakian/EWF_Reader/logger"

	UsnJrnl "github.com/aarsakian/FileSystemForensics/FS/NTFS/usnjrnl"
	"github.com/aarsakian/FileSystemForensics/disk"
	lvmlib "github.com/aarsakian/FileSystemForensics/disk/volume"
	"github.com/aarsakian/FileSystemForensics/exporter"
	"github.com/aarsakian/FileSystemForensics/filtermanager"
	"github.com/aarsakian/FileSystemForensics/filters"
	FSLogger "github.com/aarsakian/FileSystemForensics/logger"
	"github.com/aarsakian/FileSystemForensics/reporter"
	"github.com/aarsakian/FileSystemForensics/tree"
	"github.com/aarsakian/FileSystemForensics/utils"
	VMDKLogger "github.com/aarsakian/VMDK_Reader/logger"
)

func checkErr(err error, msg string) {
	if err != nil {
		log.Fatalln(msg, err)
	}
}

func main() {
	//dbmap := initDb()
	//defer dbmap.Db.Close()

	//	save2DB := flag.Bool("db", false, "bool if set an sqlite file will be created, each table will corresponed to an MFT attribute")
	var location string
	//inputfile := flag.String("MFT", "", "absolute path to the MFT file")
	evidencefile := flag.String("evidence", "", "path to image file (EWF/Raw formats are supported)")
	vmdkfile := flag.String("vmdk", "", "path to vmdk file (Sparse formats are supported)")

	flag.StringVar(&location, "location", "", "the path to export files")
	MFTSelectedEntries := flag.String("entries", "", "select file system records by entering its id, use comma as a seperator")
	showFileName := flag.String("showfilename", "", "show the name of the filename attribute of MFT records: enter (Any, Win32, Dos)")
	exportFiles := flag.String("filenames", "", "files to export use comma as a seperator")
	exportFilesPath := flag.String("path", "", "base path of files to exported must be absolute e.g. C:\\MYFILES\\ABC translates to MYFILES\\ABC")
	isResident := flag.Bool("resident", false, "check whether entry is resident")
	fromMFTEntry := flag.Int("fromentry", -1, "select file system record id to start processing")
	toMFTEntry := flag.Int("toentry", math.MaxUint32, "select file system record id to end processing")

	showRunList := flag.Bool("showrunlist", false, "show runlist of file system records")
	showFileSize := flag.Bool("filesize", false, "show file size")
	showVCNs := flag.Bool("vcns", false, "show the vcns of non resident file system attributes")
	showAttributes := flag.String("attributes", "", "show file system attributes (write any for all attributes)")
	showTimestamps := flag.Bool("showtimestamps", false, "show all file system timestamps")
	showIndex := flag.Bool("index", false, "show index structures")

	physicalDrive := flag.Int("physicaldrive", -1, "select disk drive number")
	partitionNum := flag.Int("partition", -1, "select partition number")
	physicalOffset := flag.Int("physicaloffset", -1, "offset to volume (sectors)")
	logical := flag.String("volume", "", "select directly the volume requires offset in bytes, (ntfs, lvm2)")

	buildtree := flag.Bool("tree", false, "reconstrut file system tree")

	showtree := flag.Bool("showtree", false, "show file system tree")
	showParent := flag.Bool("parent", false, "show information about parent record")
	showUsnjrnl := flag.Bool("showusn", false, "show information about NTFS usnjrnl records")
	showFull := flag.Bool("showfull", false, "show full information about record")
	showreparse := flag.Bool("showreparse", false, "show information about reparse points")

	orphans := flag.Bool("orphans", false, "show information only for orphan records")
	deleted := flag.Bool("deleted", false, "show deleted records")

	listPartitions := flag.Bool("listpartitions", false, "list partitions")
	listUnallocated := flag.Bool("listunallocated", false, "list unallocated clusters")
	fileExtensions := flag.String("extensions", "", "search file system records by extensions use comma as a seperator")
	collectUnallocated := flag.Bool("unallocated", false, "collect unallocated area of a volume")
	hashFiles := flag.String("hash", "", "hash exported files, enter md5 or sha1")
	volinfo := flag.Bool("volinfo", false, "show volume information")
	logactive := flag.Bool("log", false, "enable logging")
	showPath := flag.Bool("showpath", false, "show the full path of the selected files")
	strategy := flag.String("strategy", "overwrite", "what strategy will be used for files sharing the same name, default is ovewrite, or use Id")
	usnjrnl := flag.Bool("usnjrnl", false, "show usnjrnl information about changes to files and folders")

	flag.Parse() //ready to parse

	//var records MFT.Records
	var usnjrnlRecords UsnJrnl.Records
	var fileNamesToExport []string

	entries := utils.GetEntriesInt(*MFTSelectedEntries)

	recordsTree := tree.Tree{}

	rp := reporter.Reporter{
		ShowFileName:   *showFileName,
		ShowAttributes: *showAttributes,
		ShowTimestamps: *showTimestamps,
		IsResident:     *isResident,
		ShowFull:       *showFull,
		ShowRunList:    *showRunList,
		ShowFileSize:   *showFileSize,
		ShowVCNs:       *showVCNs,
		ShowIndex:      *showIndex,
		ShowParent:     *showParent,
		ShowPath:       *showPath,
		ShowUSNJRNL:    *showUsnjrnl,
		ShowReparse:    *showreparse,
		ShowTree:       *showtree,
	}

	if *logactive {
		now := time.Now()
		logfilename := "logs" + now.Format("2006-01-02T15_04_05") + ".txt"
		FSLogger.InitializeLogger(*logactive, logfilename)
		VMDKLogger.InitializeLogger(*logactive, logfilename)
		EWFLogger.InitializeLogger(*logactive, logfilename)

	}

	exp := exporter.Exporter{Location: location, Hash: *hashFiles, Strategy: *strategy}

	flm := filtermanager.FilterManager{}

	if *exportFiles != "" {
		fileNamesToExport = append(fileNamesToExport, utils.GetEntries(*exportFiles)...)
		flm.Register(filters.NameFilter{Filenames: fileNamesToExport})
	}

	if *fileExtensions != "" {
		flm.Register(filters.ExtensionsFilter{Extensions: strings.Split(*fileExtensions, ",")})
	}

	if *exportFilesPath != "" {
		flm.Register(filters.PathFilter{NamePath: *exportFilesPath})
	}

	if *orphans {
		flm.Register(filters.OrphansFilter{Include: *orphans})
	}

	if *deleted {
		flm.Register(filters.DeletedFilter{Include: *deleted})
	}

	if (*evidencefile != "" || *physicalDrive != -1 || *vmdkfile != "") && *logical == "" {
		disk := new(disk.Disk)
		disk.Initialize(*evidencefile, *physicalDrive, *vmdkfile)

		recordsPerPartition, err := disk.Process(*partitionNum, entries, *fromMFTEntry, *toMFTEntry)

		defer disk.Close()
		if err != nil {
			fmt.Println(err)
			return
		}

		if *usnjrnl {
			usnjrnlRecords = disk.ProcessJrnl(recordsPerPartition, *partitionNum)
		}

		if *listPartitions {
			disk.ListPartitions()
		}

		if *volinfo {
			disk.ShowVolumeInfo()
		}

		if *listUnallocated {
			disk.ListUnallocated()
		}

		if *collectUnallocated {
			exp.ExportUnallocated(*disk)
		}

		for partitionId, records := range recordsPerPartition {

			records = flm.ApplyFilters(records)

			if *usnjrnl {
				disk.ProcessJrnl(recordsPerPartition, *partitionNum)
			}

			if location != "" {
				exp.ExportRecords(records, *disk, partitionId)
				if *hashFiles != "" {
					exp.HashFiles(records)
				}
			}

			if *buildtree {
				recordsTree.Build(records)

			}

			rp.Show(records, usnjrnlRecords, partitionId, recordsTree)

		}

	} else if (*evidencefile != "" || *physicalDrive != -1 || *vmdkfile != "") && *logical == "lvm2" {

		disk := new(disk.Disk)
		disk.Initialize(*evidencefile, *physicalDrive, *vmdkfile)

		lvm2 := new(lvmlib.LVM2)
		lvm2.ProcessHeader(disk.Handler, int64(*physicalOffset*512+512))
		lvm2.Process(disk.Handler, int64(*physicalOffset*512+512), entries, *fromMFTEntry, *toMFTEntry)

	}
	/* else if *inputfile != "Disk MFT" {

		data, fsize, err := utils.ReadFile(*inputfile)
		if err != nil {
			return
		}
		var ntfs volume.NTFS

		ntfs.MFT = &MFT.MFTTable{Size: fsize}
		ntfs.ProcessMFT(data, entries, *fromMFTEntry, *toMFTEntry)

		records = flm.ApplyFilters(ntfs.MFT.Records)

		if *buildtree {
			recordsTree.Build(records)

		}

		rp.Show(records, usnjrnlRecords, 0, recordsTree)

	}
	*/
} //ends for
