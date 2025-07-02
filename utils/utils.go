package utils

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/aarsakian/FileSystemForensics/logger"
)

type NoNull string

type LargeInteger struct {
	QuadPart int64
	HighPart int32
	LowPart  int32
}

type AskedFile struct {
	Fname   string
	Id      int
	Content []byte
}

type TimeSpec struct {
	Sec  int64
	Nsec int32
}

func (timeSpec TimeSpec) ToTime() time.Time {
	return time.Unix(timeSpec.Sec, int64(timeSpec.Nsec))

}

func GetStructSize(v interface{}, size int) int {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem() // get the value pointed to
	}

	if val.Kind() == reflect.Struct {
		for i := 0; i < val.NumField(); i++ {
			field := val.Field(i) //StructField type

			switch field.Kind() {

			case reflect.Pointer:
				size += GetStructSize(field.Elem().Interface(), size)
			case reflect.Uint8:
				size += 1
			case reflect.Uint16:
				size += 2
			case reflect.Int32:
				size += 4
			case reflect.Uint32:
				size += 4
			case reflect.Uint64:
				size += 8
			case reflect.Int64:
				size += 8
			}
		}
	}

	return size

}

func ToUint32(data []byte) uint32 {
	var temp uint32
	binary.Read(bytes.NewBuffer(data), binary.LittleEndian, &temp)
	return temp
}

func ReadFile(inputfile string) ([]byte, int, error) {
	file, err := os.Open(inputfile)
	if err != nil {
		// handle the error here
		fmt.Printf("err %s in getting handle of file ", err)
		return nil, -1, err
	}
	defer file.Close()

	finfo, err := file.Stat() //file descriptor
	if err != nil {
		fmt.Printf("error getting the file size\n")
		return nil, -1, err
	}
	data := make([]byte, finfo.Size())

	_, err = file.Read(data)
	if err != nil {
		fmt.Printf("error reading  file.\n")
		return nil, -1, err
	}
	return data, int(finfo.Size()), nil
}

func GetEntries(input string) []string {

	return strings.Split(input, ",")
}

func GetEntriesInt(input string) []int {
	var entries []int
	for _, entry := range strings.Split(input, ",") {
		if entry == "" {
			continue
		}
		entryInt, _ := strconv.Atoi(entry)
		entries = append(entries, entryInt)

	}
	return entries
}

func NewLargeInteger(val int64) LargeInteger {

	return LargeInteger{QuadPart: val, HighPart: int32(val >> 32),
		LowPart: int32(val & 0xFFFFFFFF)}

}

type WindowsTime struct {
	Stamp uint64
}

func Filter[T any](s []T, f func(T) bool) []T {
	var r []T
	for _, v := range s {
		if f(v) {
			r = append(r, v)
		}
	}
	return r
}

func (winTime *WindowsTime) ConvertToIsoTime() string { //receiver winTime struct
	// t:=math.Pow((uint64(winTime.high)*2),32) + uint64(winTime.low)
	x := winTime.Stamp/10000000 - 116444736*1e2
	unixtime := time.Unix(int64(x), 0).UTC()
	return unixtime.Format("02-01-2006 15:04:05")

}

func ReadEndianInt(barray []byte) int64 {
	var buf []byte
	if len(barray) > 0 && barray[len(barray)-1]&0x80 != 0 {
		buf = []byte{0xff, 0xff, 0xff, 0xff}
	} else {
		buf = []byte{0x00, 0x00, 0x00, 0x00}
	}

	var sum int32
	copy(buf, barray)

	binary.Read(bytes.NewBuffer(buf), binary.LittleEndian, &sum)
	return int64(sum)

}

func ReadEndianUInt(barray []byte) uint64 {
	var sum uint64
	sum = 0
	for index, val := range barray {
		sum += uint64(val) << uint(index*8)

	}

	return sum
}

func RemoveNulls(val []byte) NoNull {
	var newstr strings.Builder
	for _, v := range val {
		if v != 0 {

			newstr.WriteByte(v)

		}
	}
	return NoNull(newstr.String())
}

func Hexify(barray []byte) string {

	return hex.EncodeToString(barray)

}

func Unmarshal(data []byte, v interface{}) (int, error) {
	idx := 0
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem() // get the value pointed to
	}
	if val.Kind() != reflect.Struct {
		return idx, errors.New("requires struct")
	}

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i) //StructField type
		name := val.Type().Field(i).Name
		switch field.Kind() {

		case reflect.Struct:
			switch field.Type().Name() {
			case "TimeSpec":
				var timeSpec TimeSpec
				Unmarshal(data[idx:idx+12], &timeSpec)
				field.Set(reflect.ValueOf(timeSpec))
				idx += 12

			case "WindowsTime":
				var windowsTime WindowsTime
				Unmarshal(data[idx:idx+8], &windowsTime)
				field.Set(reflect.ValueOf(windowsTime))
				idx += 8

			}

		case reflect.Pointer:
			if name == "Key" || name == "DropProgress" {
				idx += 17
			}

		case reflect.Uint8:
			var temp uint8
			binary.Read(bytes.NewBuffer(data[idx:idx+1]), binary.LittleEndian, &temp)
			field.SetUint(uint64(temp))
			idx += 1
		case reflect.Int16:
			var temp int16
			binary.Read(bytes.NewBuffer(data[idx:idx+2]), binary.LittleEndian, &temp)
			field.SetInt(int64(temp))
			idx += 2
		case reflect.Uint16:
			var temp uint16
			binary.Read(bytes.NewBuffer(data[idx:idx+2]), binary.LittleEndian, &temp)
			field.SetUint(uint64(temp))
			idx += 2
		case reflect.Uint32:
			var temp uint32
			binary.Read(bytes.NewBuffer(data[idx:idx+4]), binary.LittleEndian, &temp)
			field.SetUint(uint64(temp))
			idx += 4
		case reflect.Int64:
			var temp int64
			binary.Read(bytes.NewBuffer(data[idx:idx+8]), binary.LittleEndian, &temp)
			idx += 8
			field.SetInt(temp)
		case reflect.Uint64:
			var temp uint64

			if name == "ParRef" || name == "EntryRef" {
				buf := make([]byte, 8)
				copy(buf, data[idx:idx+6])
				binary.Read(bytes.NewBuffer(buf), binary.LittleEndian, &temp)
				idx += 6
			} else if name == "ChildVCN" {
				len := val.Elem().FieldByName("Len").Uint()
				binary.Read(bytes.NewBuffer(data[len-8:len]), binary.LittleEndian, &temp)

			} else {
				binary.Read(bytes.NewBuffer(data[idx:idx+8]), binary.LittleEndian, &temp)
				idx += 8
			}
			field.SetUint(temp)
		case reflect.Bool:
			field.SetBool(false)
			idx += 1
		case reflect.Array:
			arrT := reflect.ArrayOf(field.Len(), reflect.TypeOf(data[0])) //create array type to hold the slice
			arr := reflect.New(arrT).Elem()                               //initialize and access array
			var end int
			if idx+field.Len() > len(data) { //determine end
				end = len(data)
			} else {
				end = idx + field.Len()
			}
			if idx > end {
				return idx, errors.New("exceede available buffer")
			}
			for idx, val := range data[idx:end] {

				arr.Index(idx).Set(reflect.ValueOf(val))
			}

			field.Set(arr)
			idx += field.Len()

		}

	}
	return idx, nil
}

func Bytereverse(barray []byte) []byte { //work with indexes
	//  fmt.Println("before",barray)
	for i, j := 0, len(barray)-1; i < j; i, j = i+1, j-1 {

		barray[i], barray[j] = barray[j], barray[i]

	}
	return barray

}

func WriteFile(filename string, content []byte) {
	if file, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644); err != nil {
		fmt.Printf("err %s opening the file \n", err)
	} else {
		bytesWritten, err := file.Write(content)
		if err != nil {
			fmt.Printf("err %s writing the file \n", err)

		}

		msg := fmt.Sprintf("wrote file %s  %d bytes",
			filename, bytesWritten)
		logger.FSLogger.Info(msg)
		fmt.Printf(msg + "\n")
		file.Close()
	}

}

func DecodeUTF16(b []byte) string {
	utf := make([]uint16, (len(b)+(2-1))/2) //2 bytes for one char?
	for i := 0; i+(2-1) < len(b); i += 2 {
		utf[i/2] = binary.LittleEndian.Uint16(b[i:])
	}
	if len(b)/2 < len(utf) {
		utf[len(utf)-1] = utf8.RuneError
	}
	return string(utf16.Decode(utf))

}

func WriteToCSV(file *os.File, data string) {
	_, err := file.WriteString(data)
	if err != nil {
		// handle the error here
		fmt.Printf("err %s\n", err)
		return
	}
}

func readEndianU(barray []byte) (val interface{}) {
	//conversion function
	//fmt.Println("before conversion----------------",barray)
	//fmt.Printf("len%d ",len(barray))

	switch len(barray) {
	case 8:
		var vale uint64
		binary.Read(bytes.NewBuffer(barray), binary.LittleEndian, &vale)
		val = vale
	case 6:

		var vale uint32
		buf := make([]byte, 6)
		binary.Read(bytes.NewBuffer(barray[:4]), binary.LittleEndian, &vale)
		var vale1 uint16
		binary.Read(bytes.NewBuffer(barray[4:]), binary.LittleEndian, &vale1)
		binary.LittleEndian.PutUint32(buf[:4], vale)
		binary.LittleEndian.PutUint16(buf[4:], vale1)
		val, _ = binary.ReadUvarint(bytes.NewBuffer(buf))

	case 4:
		var vale uint32
		//   fmt.Println("barray",barray)
		binary.Read(bytes.NewBuffer(barray), binary.LittleEndian, &vale)
		val = vale
	case 2:

		var vale uint16

		binary.Read(bytes.NewBuffer(barray), binary.LittleEndian, &vale)
		//   fmt.Println("after conversion vale----------------",barray,vale)
		val = vale

	case 1:

		var vale uint8

		binary.Read(bytes.NewBuffer(barray), binary.LittleEndian, &vale)
		//      fmt.Println("after conversion vale----------------",barray,vale)
		val = vale

	default: //best it would be nil
		var vale uint64

		binary.Read(bytes.NewBuffer(barray), binary.LittleEndian, &vale)
		val = vale
	}

	return val
}

func readEndianFloat(barray []byte) (val uint64) {

	//    fmt.Printf("len%d ",len(barray))

	binary.Read(bytes.NewBuffer(barray), binary.LittleEndian, &val)
	return val
}

func readEndianString(barray []byte) (val []byte) {

	binary.Read(bytes.NewBuffer(barray), binary.LittleEndian, &val)

	return val
}

func FindEvidenceFiles(path_ string) []string {

	basePath := filepath.Dir(path_)

	files, err := os.ReadDir(basePath)
	if err != nil {
		log.Fatal("ERR", err)
	}

	var filenames []string
	r, _ := regexp.Compile("\\.[eE][a-zA-Z0-9]{1,2}$")
	for _, finfo := range files {

		if finfo.IsDir() {

			continue
		}

		if r.MatchString(finfo.Name()) {

			filenames = append(filenames, filepath.Join(basePath, finfo.Name())) //supply channel
			//fmt.Println("INFO", basePath+finfo.Name(), strings.Split(fname, ".")[0])

		}
	}

	return filenames

}

func SetProgress(progressStat int, msg string) {
	clearLine := "\x1B[2K"
	io.WriteString(os.Stdout, clearLine)
	eraseCursor := "\x1B[0J"
	io.WriteString(os.Stdout, eraseCursor)
	str := fmt.Sprintf("%4d%% %s", progressStat, msg)

	io.WriteString(os.Stdout, str)
}

func StringifyGUID(barray []byte) string {
	return fmt.Sprintf("%x-%x-%x-%x-%x", Bytereverse(barray[0:4]),
		Bytereverse(barray[4:6]), Bytereverse(barray[6:8]),
		barray[8:10], barray[10:])
}

func StringifyUUID(barray []byte) string {
	return fmt.Sprintf("%x-%x-%x-%x-%x", barray[0:4],
		barray[4:6], barray[6:8],
		barray[8:10], barray[10:])
}
