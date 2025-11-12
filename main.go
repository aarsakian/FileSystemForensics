package main

import (
	//"C"

	//"database/sql"

	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	_ "net/http/pprof"
	"strconv"
	"strings"
	"time"

	EWFLogger "github.com/aarsakian/EWF_Reader/logger"
	metadata "github.com/aarsakian/FileSystemForensics/FS"
	vssLib "github.com/aarsakian/FileSystemForensics/FS/NTFS/VSS"
	UsnJrnl "github.com/aarsakian/FileSystemForensics/FS/NTFS/usnjrnl"
	"github.com/aarsakian/FileSystemForensics/disk"
	Vol "github.com/aarsakian/FileSystemForensics/disk/volume"
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
	mftOffset := flag.Int("mftoffset", 0, "physical offset to the  $MFT file")
	evidencefile := flag.String("evidence", "", "path to image file (EWF/Raw formats are supported)")
	vmdkfile := flag.String("vmdk", "", "path to vmdk file (Sparse formats are supported)")

	flag.StringVar(&location, "location", "", "the path to export files")
	MFTSelectedEntries := flag.String("entries", "", "select file system records by entering its id, use comma as a seperator")
	showFileName := flag.String("showfilename", "", "show the name of the filename attribute of MFT records: enter (Any, Win32, Dos)")
	exportFiles := flag.String("filenames", "", "files to export use comma as a seperator")
	exportFilesPath := flag.String("path", "", "base path of files to exported must be absolute e.g. C:\\MYFILES\\ABC translates to MYFILES\\ABC")
	isResident := flag.Bool("resident", false, "check whether entry is resident")
	fromMFTEntry := flag.Int("fromentry", 0, "select file system record id to start processing")
	toMFTEntry := flag.Int("toentry", math.MaxUint32, "select file system record id to end processing")

	showRunList := flag.Bool("showrunlist", false, "show runlist of file system records")
	showFileSize := flag.Bool("filesize", false, "show file size")
	showVCNs := flag.Bool("vcns", false, "show the vcns of non resident file system attributes")
	showAttributes := flag.String("attributes", "", "show file system attributes (write any for all attributes)")
	showTimestamps := flag.Bool("showtimestamps", false, "show all file system timestamps")
	showIndex := flag.Bool("showindex", false, "show index structures")

	physicalDrive := flag.Int("physicaldrive", -1, "select disk drive number")
	partitionNum := flag.Int("partition", 0, "select partition number")
	physicalOffset := flag.Int("physicaloffset", -1, "offset to volume (sectors)")
	logical := flag.String("volume", "", "select directly the volume requires offset in bytes, (ntfs, lvm2)")
	searchFS := flag.String("searchfs", "", "look for traces of the file system (NTFS is supported)")
	searchOffset := flag.Int("searchoffset", 0, "offset in bytes to search for file system structures")
	buildtree := flag.Bool("tree", false, "reconstrut file system tree")

	showtree := flag.Bool("showtree", false, "show file system tree")
	showParent := flag.Bool("parent", false, "show information about parent record")
	showUsnjrnl := flag.Bool("showusn", false, "show information about NTFS usnjrnl records")
	showFull := flag.Bool("showfull", false, "show full information about record")
	showreparse := flag.Bool("showreparse", false, "show information about reparse points")

	clusters := flag.String("clusters", "", "clusters to look for")
	listvss := flag.Bool("listvss", false, "list vss copied clusters")

	benchmark := flag.Bool("benchmark", false, "test HD speed")
	showVSSClusters := flag.Bool("showvssclusters", false, "show volume shadow releveat information for selected clusters")

	orphans := flag.Bool("orphans", false, "show information only for orphan records")
	deleted := flag.Bool("deleted", false, "show deleted records")
	vss := flag.Bool("vss", false, "process shadow volume copies")
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
	logfile := flag.Bool("logfile", false, "parse and show $logfile")

	profile := flag.Bool("profile", false, "profile memory usage")

	flag.Parse() //ready to parse

	//var records MFT.Records
	var usnjrnlRecords UsnJrnl.Records
	var shadowVolume vssLib.ShadowVolume
	var fileNamesToExport []string

	var err error
	var recordsPerPartition map[int][]metadata.Record

	entries := utils.GetEntriesInt(*MFTSelectedEntries)

	recordsTree := tree.Tree{}
	if *profile {
		go func() {
			log.Println("pprof listening on :6060")
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	rp := reporter.Reporter{
		ShowFileName:    *showFileName,
		ShowAttributes:  *showAttributes,
		ShowTimestamps:  *showTimestamps,
		IsResident:      *isResident,
		ShowFull:        *showFull,
		ShowRunList:     *showRunList,
		ShowFileSize:    *showFileSize,
		ShowVCNs:        *showVCNs,
		ShowIndex:       *showIndex,
		ShowParent:      *showParent,
		ShowPath:        *showPath,
		ShowUSNJRNL:     *showUsnjrnl,
		ShowReparse:     *showreparse,
		ShowTree:        *showtree,
		ShowVSSClusters: *showVSSClusters,
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
		dsk := new(disk.Disk)
		dsk.Initialize(*evidencefile, *physicalDrive, *vmdkfile)
		defer dsk.Close()

		if *benchmark {
			dsk.Benchmark()
		}

		if *mftOffset != 0 {
			vol := Vol.NTFS{}
			vol.VBR = &Vol.VBR{BytesPerSector: 512, SectorsPerCluster: uint8(4)}
			vol.Process(dsk.Handler, int64(*mftOffset), entries, *fromMFTEntry, *toMFTEntry)

		}

		if *searchFS != "" {
			recordsPerPartition = dsk.SearchFileSystemCH(*searchFS, *searchOffset)

		} else {
			recordsPerPartition, err = dsk.Process(*partitionNum-1, entries, *fromMFTEntry, *toMFTEntry)
			if err != nil {
				fmt.Println(err)
				return
			}

		}

		if *usnjrnl {
			usnjrnlRecords = dsk.ProcessJrnl(recordsPerPartition, *partitionNum-1)
		}

		if *logfile {
			dsk.ProcessLogFile(recordsPerPartition, *partitionNum-1)
		}

		if *vss {
			shadowVolume = dsk.ProcessVSS(*partitionNum - 1)
		}

		if *listPartitions {
			dsk.ListPartitions()
		}

		if *volinfo {
			dsk.ShowVolumeInfo()
		}

		if *listUnallocated {
			dsk.ListUnallocated()
		}

		if *collectUnallocated {
			exp.ExportUnallocated(*dsk)
		}

		if *clusters != "" {
			var _clusters []int
			for _, cluster := range strings.Split(*clusters, ",") {
				val, _ := strconv.ParseUint(cluster, 10, 64)
				_clusters = append(_clusters, int(val))
			}
			offsets := shadowVolume.GetClustersInfo(_clusters)
			for idx, offset := range offsets {
				if offset == -1 {
					fmt.Printf("%d cl shadow offset not found\n", _clusters[idx])
				} else {
					fmt.Printf("%d cl shadow offset cl %d\n", _clusters[idx], offset)
				}
			}

		}

		if *listvss {
			shadowVolume.ListVSS()
		}

		for partitionId, records := range recordsPerPartition {

			records = flm.ApplyFilters(records)

			if *usnjrnl {
				dsk.ProcessJrnl(recordsPerPartition, partitionId)
			}

			if location != "" {
				exp.ExportRecords(records, *dsk, partitionId)
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

		dsk2 := new(disk.Disk)
		dsk2.Initialize(*evidencefile, *physicalDrive, *vmdkfile)

		defer dsk2.Close()

		lvm2 := new(Vol.LVM2)
		err := lvm2.ProcessHeader(dsk2.Handler, int64(*physicalOffset*512))
		if err == nil {
			lvm2.Process(dsk2.Handler, int64(*physicalOffset*512), entries, *fromMFTEntry, *toMFTEntry)
		}

	}

}
