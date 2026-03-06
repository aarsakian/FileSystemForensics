module github.com/aarsakian/FileSystemForensics

go 1.18

require golang.org/x/sys v0.12.0

require (
	github.com/aarsakian/VMDK_Reader v0.0.0-20240910071554-9d72aac7f6b9
	golang.org/x/text v0.13.0
)

require github.com/aarsakian/EWF_Reader v0.0.0-20260306182323-6ede308af4dd // indirect

replace github.com/aarsakian/EWF_Reader => ..\..\go\pkg\mod\github.com\aarsakian\!e!w!f_!reader@v0.0.0-20251219163715-06f27839d852
