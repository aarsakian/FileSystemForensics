FileSystemForensics
============

### a tool to inspect, extract files and file system metadata. It currently supports NTFS, BTRFS, and BitLocker-encrypted volumes.



By using this tool, you can explore NTFS and its file system attributes. You can selectively extract filesystem information of a record, or for a range of records. In addition, you can export the contents of files.

Exporting files can be achieved either by mounting the evidence and providing its physical drive order and partition number or by using the acquired forensic image (Expert Witness Format), or a virtual machine disk format (VMDK) as input.

For BitLocker volumes, provide the volume image/device together with `-password` or `-recoverykey` to unlock and process the encrypted volume.

#### Examples #####
you can explore NTFS or BTRFS by providing physical drive number and partition number 

e.g. *-physicaldrive 0 -partition 1* translates to \\\\.\\PHYSICALDRIVE0 D drive respectively,


or by using as input an expert witness format image 

e.g. *-evidence path_to_evidence -partition 1*.

To filter exported records to those whose file headers match known signatures, add:

e.g. *-evidence path_to_evidence -partition 1 -verifysignatures strict*.

##### Usage information #####

The current CLI flags can be reviewed with:

  go run . --help

Flags are grouped by purpose:

- Input and target selection: -evidence, -physicaldrive, -partition, -volume, -physicaloffset, and -mftoffset
- Record selection and filtering: -entries, -fromentry, -toentry, -orphans, -deleted, -extensions, -filenames, -path, and -verifysignatures
- Display and reporting: -showfilename, -showfull, -showtree, -showpath, -showtimestamps, -showrunlist, -showvcns, -showvssclusters, -showclusters, -showbitlocker, -showparent, -showattributes, -showfilesize, -showindex, -volinfo, -tree, -usnjrnl, and -logfile
- Export and hashing: -export, -recreatepath, -strategy, -hash, -unallocated, -listunallocated, -listpartitions, and -resident
- BitLocker, VSS, and diagnostics: -password, -recoverykey, -vss, -listvss, -log, -benchmark, and -profile

Current options include:

  -benchmark
        test HD speed

  -clusters string
        clusters to look for

  -deleted
        show only deleted records

  -entries string
        select file system records by entering its id, use comma as a seperator

  -evidence string
        path to image file (EWF/VHDX/VMDK/Raw formats are supported)

  -export string
        the path to export files

  -extensions string
        search file system records by extensions use comma as a seperator

  -filenames string
        files to export use comma as a seperator

  -fromentry int
        select file system record id to start processing

  -hash string
        hash exported files, enter md5 or sha1

  -listpartitions
        list partitions

  -listunallocated
        list unallocated clusters

  -listvss
        list vss copied clusters

  -log
        enable logging

  -logfile
        parse and show $logfile

  -mftoffset int
        physical offset to the  $MFT file

  -orphans
        show information only for orphan records

  -partition int
        select partition number

  -password string
        password for Bitlocker volumes

  -path string
        base path of files to exported must be absolute e.g. C:\MYFILES\ABC translates to MYFILES\ABC

  -physicaldrive int
        select disk drive number (default -1)

  -physicaloffset int
        offset to volume (sectors) (default -1)

  -profile
        profile memory usage

  -recoverykey string
        recovery key for Bitlocker volumes

  -recreatepath
        recreate file path

  -resident
        check whether has resident data attribute

  -searchfs string
        look for traces of the file system (NTFS is supported)

  -searchoffset int
        offset in bytes to search for file system structures

  -showattributes string
        show file system attributes (write any for all attributes, use comma for more than one attributes),

  -showbitlocker
        show information about bitlocker volume

  -showclusters
        show allocated clusters of a record inside shadow volumes

  -showfilename
        show the name of a file or a directory

  -showfilesize
        show file size

  -showfull
        show full information about record

  -showindex
        show index structures

  -showparent
        show information about parent record

  -showpath
        show the full path of the selected files

  -showrunlist
        show runlist of file system records

  -showtimestamps
        show all file system timestamps

  -showtree
        show file system tree

  -showusn
        show information about NTFS usnjrnl records

  -showvcns
        show the vcns of non resident file system attributes

  -showvssclusters
        show volume shadow relevant information for selected records

  -strategy string
        what strategy will be used for files sharing the same name, default is ovewrite, or use Id (default "overwrite")

  -toentry int
        select file system record id to end processing (default 4294967295)

  -tree
        reconstrut file system tree

  -unallocated
        collect unallocated area of a volume

  -usnjrnl
        show usnjrnl information about changes to files and folders

  -verifysignatures string
        verify file system records by file signatures, non verified records will be omitted, allowed values are strict|permissive. (strict filters out mismatched extensions) (check signatures.csv for the list of files)

  -volinfo
        show volume information

  -volume string
        select directly the volume requires offset in bytes, (ntfs, lvm2)

  -vss
        process shadow volume copies
