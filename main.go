package main

import (
	"fmt"
	"os"
	"time"
)

const (
	TIME_HHMMSS = 8
)

func getFirstTimeFromBuffer(buf []byte) (string, int64) {
	for index, c := range buf {
		if c == '\n' && (index+1+TIME_HHMMSS) <= len(buf) {
			return string(buf[index+1 : index+1+TIME_HHMMSS]), int64(index + 1)
		}
	}
	return "", 0
}

func getLastTimeFromBuffer(buf []byte) (string, int64) {
	for index := len(buf) - 1; index >= 0; index-- { //, c := range buf {
		c := buf[index]
		if c == '\n' && (index+1+TIME_HHMMSS) <= len(buf) {
			return string(buf[index+1 : index+1+TIME_HHMMSS]), int64(index + 1)
		}
	}
	return "", 0
}

func getAfterTime(buf []byte, to time.Time) int {
	for index, c := range buf {
		if c == '\n' && (index+1+TIME_HHMMSS) <= len(buf) {
			strtime := string(buf[index+1 : index+1+TIME_HHMMSS])
			at, _ := time.Parse("15:04:05", strtime)
			if at.Sub(to) >= 0 {
				return index+1
			}
		}
	}
	return len(buf)
}

func main() {
	BLOCK_SIZE_INSPECTION := int64(512)
	BLOCK_SIZE_READ := int64(4096)
	MAX_ITERS := 1000

	args := os.Args[1:]
	if len(args) < 3 {
		fmt.Println("Usage: logcat file.log start_time(HH:MM) end_time(HH:MM)")
		return
	}
	from, ef := time.Parse("15:04", args[1]) //"14:00")
	to, et := time.Parse("15:04", args[2])   //"14:05")
	if ef != nil || et != nil {
		panic(fmt.Errorf("Error converting time args (%v) (%v)",
			ef, et,
		))
	} else if from.Sub(to) > 0 {
		panic(fmt.Errorf("start(%v) must be before end(%v)", args[1], args[2]))
	}
	if f, err := os.Open(args[0]); err != nil {
		panic(err)
	} else {
		b := make([]byte, BLOCK_SIZE_INSPECTION)
		info, _ := f.Stat()
		fileSize := info.Size()

		pos := fileSize / 2
		increment := pos / 2
		for iter := 0; iter < MAX_ITERS; iter++ {
			f.ReadAt(b, pos)
			logtime, _ := getFirstTimeFromBuffer(b)
			if len(logtime) != 0 {
				at, _ := time.Parse("15:04:05", string(logtime))
				tdiff := at.Sub(from)
				fmt.Fprintln(os.Stderr, logtime, tdiff)
				if tdiff < 0 && tdiff > time.Minute*-1 {
					offset := 0
					for offset = getAfterTime(b, from); offset == len(b); offset = getAfterTime(b, from){
						pos += BLOCK_SIZE_INSPECTION
						f.ReadAt(b, pos)
					}
					pos += int64(offset)
					break
				} else if at.Sub(from) > 0 {
					//before
					//fmt.Println(pos, pos - increment)
					pos -= increment
					increment /= 2
				} else {
					//fmt.Println(pos, pos + increment)
					pos += increment
					increment /= 2
				}
			} else {
				//if you can't find a \n in the current block, advance to the next block minus the length of the timestamp
				//which covers the case of partial timestamp between the blocks
				pos += BLOCK_SIZE_INSPECTION - TIME_HHMMSS
			}
		}

		b = make([]byte, BLOCK_SIZE_READ)
		for {
			f.ReadAt(b, pos)
			logtime, _ := getLastTimeFromBuffer(b)
			if logtime != "" {
				at, _ := time.Parse("15:04:05", logtime)
				//fmt.Fprintln(os.Stderr, at)
				if at.Sub(to) > 0 {
					fmt.Printf("%s", b[:getAfterTime(b, to)])
					break
				} else {
					fmt.Printf("%s", b)
				}
			} else {
				//no new lines in this segment, just print out, and then deal with it later
				fmt.Printf("%s", b)
			}
			pos += BLOCK_SIZE_READ
		}
	}
}
