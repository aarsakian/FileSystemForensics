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

e.g. *-evidence path_to_evidence -partition 1 -verifysignatures*.

##### Usage information #####

The current CLI flags can be reviewed with:

  go run . --help

Common options include:

  -attributes string
        show file system attributes (write any for all attributes)

  -benchmark
        test HD speed

  -clusters string
        clusters to look for

  -deleted
        show deleted records

  -entries string
        select file system records by entering their ID; use a comma as a separator

  -evidence string
        path to image file (EWF/VHDX/Raw formats are supported)

  -export string
        the path to export files

  -extensions string
        search file system records by extension; use a comma as a separator

  -filenames string
        files to export; use a comma as a separator

  -filesize
        show file size

  -fromentry int
        select file system record ID to start processing (default 0)

  -hash string
        hash exported files; enter md5 or sha1

  -listpartitions
        list partitions

  -listunallocated
        list unallocated clusters

  -listvss
        list VSS copied clusters

  -log
        enable logging

  -logfile
        parse and show $logfile

  -mftoffset int
        physical offset to the $MFT file

  -orphans
        show information only for orphan records

  -parent
        show information about parent record

  -partition int
        select partition number

  -password string
        password for BitLocker volumes

  -path string
        base path of files to export; must be absolute, e.g. C:\MYFILES\ABC translates to MYFILES\ABC

  -physicaldrive int
        select disk drive number (default -1)

  -physicaloffset int
        offset to volume (sectors) (default -1)

  -profile
        profile memory usage

  -recoverykey string
        recovery key for BitLocker volumes

  -recreatepath
        recreate file path when exporting files

  -resident
        check whether entry is resident

  -searchfs string
        look for traces of the file system (NTFS is supported)

  -searchoffset int
        offset in bytes to search for file system structures

  -showbitlocker
        show information about BitLocker volume

  -showclusters
        show allocated clusters of a record inside shadow volumes

  -showfilename string
        show the name of the filename attribute of MFT records: enter Any, Win32, or Dos

  -showfull
        show full information about record

  -showindex
        show index structures

  -showpath
        show the full path of the selected files

  -showreparse
        show information about reparse points

  -showrunlist
        show runlist of file system records

  -showtimestamps
        show all file system timestamps

  -showtree
        show file system tree

  -showusn
        show information about NTFS usnjrnl records

  -showvssclusters
        show volume shadow relevant information for selected records

  -strategy string
        what strategy will be used for files sharing the same name; default is overwrite, or use Id (default "overwrite")

  -toentry int
        select file system record ID to end processing (default 4294967295)

  -tree
        reconstruct file system tree

  -unallocated
        collect unallocated area of a volume

  -usnjrnl
        show usnjrnl information about changes to files and folders

  -vcns
        show the vcns of non-resident file system attributes

  -vmdk string
        path to VMDK file (Sparse formats are supported)

  -verifysignatures string
        verify file system records by file signatures; allowed values are strict|permissive
        (strict filters out mismatched extensions; check signatures/signatures.csv for the list of files)

  -volinfo
        show volume information

  -volume string
        select directly the volume; requires offset in bytes (ntfs, lvm2)

  -vss
        process shadow volume copies
