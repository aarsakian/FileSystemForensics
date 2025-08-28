FileSystemForensics
============

### a tool to inspect, extract, files and file system metadata. It currently supports NTFS,  BTRFS. 



By using this tool, you can explore NTFS and its file system attributes. You can selectively extract filesystem information of a record, or for a range of records. In addition, you can export the contents of files. 

Exporting files can be achieved either by mounting the evidence and providing its physical drive order and partition number or by using the acquired forensic image (Expert Witness Format), or a virtual machine disk format (VMDK). 

#### Examples #####
you can explore NTFS or BTRFS by providing physical drive number and partition number 

e.g. *-physicaldrive 0 -partition 1* translates to \\\\.\\PHYSICALDRIVE0 D drive respectively,


or by using as input an expert witness format image 

e.g. *-evidence path_to_evidence -partition 1*.

##### Usage information  type: FileSystemForensics.exe -h #####

 
  -attributes string
        show file system attributes (write any for all attributes)
 
  -deleted
        show deleted records
 
  -entries string
        select file system records by entering its id, use comma as a seperator
 
  -evidence string
        path to image file (EWF/Raw formats are supported)
 
  -extensions string
        search file system records by extensions use comma as a seperator
 
  -filenames string
        files to export use comma as a seperator
 
  -filesize
        show file size
 
  -fromentry int
        select file system record id to start processing (default -1)
 
  -hash string
        hash exported files, enter md5 or sha1
 
  -listpartitions
        list partitions
 
  -listunallocated
        list unallocated clusters
 
  -location string
        the path to export files
 
  -log
        enable logging
 
  -orphans
        show information only for orphan records
 
  -parent
        show information about parent record
 
  -partition int
        select partition number
 
  -path string
        base path of files to exported must be absolute e.g. C:\MYFILES\ABC translates to MYFILES\ABC
 
  -physicaldrive int
        select disk drive number (default -1)

  -physicaloffset int
        offset to volume (sectors) (default -1)

  -resident
        check whether entry is resident

  -showfilename string
        show the name of the filename attribute of MFT records: enter (Any, Win32, Dos)

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

  -vcns
        show the vcns of non resident file system attributes

  -vmdk string
        path to vmdk file (Sparse formats are supported)

  -volinfo
        show volume information

  -volume string
        select directly the volume requires offset in bytes, (ntfs, lvm2)
