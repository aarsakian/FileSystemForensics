package exporter

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	metadata "github.com/aarsakian/FileSystemForensics/FS"
	"github.com/aarsakian/FileSystemForensics/disk"
	"github.com/aarsakian/FileSystemForensics/logger"
	"github.com/aarsakian/FileSystemForensics/utils"
)

type Exporter struct {
	Location string
	Hash     string
	Strategy string
}

func (exp Exporter) ExportData(wg *sync.WaitGroup, results <-chan utils.AskedFile) {
	defer wg.Done()

	for result := range results {
		if exp.Strategy == "Id" {

			exp.CreateFile(fmt.Sprintf("[%d]%s", result.Id, result.Fname), result.Content)

		} else {
			exp.CreateFile(result.Fname, result.Content)
		}

	}

}

func (exp Exporter) ExportUnallocated(physicalDisk disk.Disk) {

	blocks := make(chan []byte) // write for consecutive blocks
	go physicalDisk.CollectedUnallocated(blocks)
	fullpath := filepath.Join(exp.Location, "Unallocated")
	for block := range blocks {
		utils.WriteFile(fullpath, block)
	}
}

func (exp Exporter) SetFilesToLogicalSize(records []metadata.Record) {
	var fname string
	for _, record := range records {
		if exp.Strategy == "Id" {
			fname = fmt.Sprintf("[%d]%s", record.GetID(), record.GetFname())
		} else {
			fname = record.GetFname()
		}

		e := os.Truncate(filepath.Join(exp.Location, fname), record.GetLogicalFileSize())
		if e != nil {
			fmt.Printf("Error truncating %s\n", e)
		}

	}
}

func (exp Exporter) ExportRecords(records []metadata.Record, physicalDisk disk.Disk, partitionNum int) {
	if exp.Location == "" {
		msg := "No export location was set"
		logger.FSLogger.Warning(msg)
		fmt.Printf("%s \n", msg)
		return
	}

	if len(records) == 0 {
		msg := "No records  found in Partition"
		logger.FSLogger.Warning(msg)
		fmt.Printf("%s \n", msg)
		return
	}

	fmt.Printf("About to export %d files\n", len(records))
	results := make(chan utils.AskedFile, len(records))

	wg := new(sync.WaitGroup)
	wg.Add(2)

	go physicalDisk.Worker(wg, records, results, partitionNum) //producer
	go exp.ExportData(wg, results)                             //pipeline copies channel

	wg.Wait()
}

func (exp Exporter) HashFiles(records []metadata.Record) {

	if exp.Hash != "MD5" && exp.Hash != "SHA1" {
		fmt.Printf("Only Supported Hashes are MD5 or SHA1 and not %s!\n", exp.Hash)
		return
	}
	fmt.Printf("Hashing Stage\n")
	for _, record := range records {
		fname := record.GetFname()

		data, e := os.ReadFile(filepath.Join(exp.Location, fname))
		if e != nil {
			fmt.Printf("ERROR %s", e)
			continue
		}
		if exp.Hash == "MD5" {
			fmt.Printf("File %s has %s %s \n", fname, exp.Hash, utils.GetMD5(data))
		} else if exp.Hash == "SHA1" {
			fmt.Printf("File %s has %s %s \n", fname, exp.Hash, utils.GetSHA1(data))
		}

	}

}

func (exp Exporter) CreateFile(fname string, data []byte) {
	var err error
	fullpath := filepath.Join(exp.Location, fname)
	err = os.MkdirAll(exp.Location, 0750)
	if err != nil && !os.IsExist(err) {
		fmt.Println(err)
	}
	if _, err = os.Stat(fullpath); !errors.Is(err, os.ErrNotExist) {
		err = os.Remove(fullpath)
		if err != nil && !os.IsExist(err) {
			fmt.Println("SDSD", fullpath, err)
		}

	}

	utils.WriteFile(fullpath, data)

}
