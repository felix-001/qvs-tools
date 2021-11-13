package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

const (
	extinf = "#EXTINF:"
)

var (
	ErrNotFound = errors.New("not found")
)

type M3u8Parser struct {
	lastEnd           int64
	start             int64
	end               int64
	duration          int64
	totalDuration     int64
	realTotalDuration int64
	firstTSStart      int64
	totalGap          int64
	maxGap            int64
	minGap            int64
	lineNum           int
}

func New() *M3u8Parser {
	return &M3u8Parser{
		lastEnd:       0,
		lineNum:       1,
		totalDuration: 0,
		firstTSStart:  0,
		maxGap:        0,
		minGap:        0,
		totalGap:      0}
}

func (self *M3u8Parser) getDuration(line string) (int64, error) {
	start := strings.Index(line, extinf) + len(extinf)
	if start == -1 {
		log.Println("cant found", extinf)
		return 0, ErrNotFound
	}
	end := strings.Index(line, ",")
	if start == -1 {
		log.Println("cant found ,")
		return 0, ErrNotFound
	}
	s := line[start:end]
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	return int64(v * 1000), nil
}

func (self *M3u8Parser) getTimestamp(line string) (int64, int64, error) {
	endPos := strings.Index(line, ".ts")
	if endPos == -1 {
		log.Println("cat found .ts")
		return 0, 0, ErrNotFound
	}
	lastEndPos := endPos
	startPos := strings.LastIndex(line[:endPos], "/")
	if startPos == -1 {
		log.Println("cat found /")
		return 0, 0, ErrNotFound
	}
	endPos = strings.Index(line, "-")
	if endPos == -1 {
		log.Println("cat found -")
		return 0, 0, ErrNotFound
	}
	startStr := line[startPos+1 : endPos]
	start, err := strconv.ParseInt(startStr, 10, 64)
	if err != nil {
		log.Println("parse int err", err, startStr)
		return 0, 0, err
	}
	endStr := line[endPos+1 : lastEndPos]
	end, err := strconv.ParseInt(endStr, 10, 64)
	if err != nil {
		log.Println("parse int err", err, endStr)
		return 0, 0, err
	}

	return start, end, nil
}

func (self *M3u8Parser) check(line string) {
	if self.start == 0 || self.end == 0 || self.duration == 0 {
		return
	}
	if (self.end - self.start) != self.duration {
		log.Println("lineNum:", self.lineNum, "ts开始结束时间与EXTINF不符", line,
			"duration:", self.duration, "real:", self.end-self.start,
			"start:", self.start, "end:", self.end)
	}
	if (self.lastEnd != 0) && (self.lastEnd/1000 != self.start/1000) {
		log.Println("lineNum:", self.lineNum, "当前ts的时间戳与上一个ts时间戳不连续",
			"lastEnd:", self.lastEnd, "start:", self.start)
	}
	if self.lastEnd != 0 {
		gap := self.start - self.lastEnd
		self.totalGap += gap
		if self.maxGap < gap {
			self.maxGap = gap
		}
		if self.minGap == 0 {
			self.minGap = gap
		} else if self.minGap > gap {
			self.minGap = gap
		}
	}
	self.lastEnd = self.end
	self.start = 0
	self.end = 0
	self.duration = 0
}

func (self *M3u8Parser) parseLine(line string) error {
	if strings.Contains(line, extinf) {
		var err error
		self.duration, err = self.getDuration(line)
		if err != nil {
			return err
		}
		self.totalDuration += self.duration
	}
	if strings.Contains(line, "/v1/record/ts/") {
		var err error
		self.start, self.end, err = self.getTimestamp(line)
		if err != nil {
			return err
		}
		if self.firstTSStart == 0 {
			self.firstTSStart = self.start
		}
		self.realTotalDuration = self.end - self.firstTSStart
	}
	self.check(line)

	return nil
}

func (self *M3u8Parser) analysis(m3u8 string) error {
	scanner := bufio.NewScanner(strings.NewReader(m3u8))
	for scanner.Scan() {
		if err := self.parseLine(scanner.Text()); err != nil {
			return err
		}
		self.lineNum++
	}
	return nil
}

func (self *M3u8Parser) dump() {
	fmt.Println("total duration:", self.totalDuration)
	fmt.Println("real total duration:", self.realTotalDuration)
	fmt.Println("gap duraion:", self.realTotalDuration-self.totalDuration)
	fmt.Println("total gap:", self.totalGap)
	fmt.Println("min gap:", self.minGap)
	fmt.Println("max gap:", self.maxGap)
}

func main() {
	log.SetFlags(log.Lshortfile)
	file := flag.String("file", "", "input file")
	flag.Parse()
	if *file == "" {
		log.Println("no input file")
		return
	}
	b, err := ioutil.ReadFile(*file)
	if err != nil {
		log.Println("open file", *file, "err", err)
		return
	}
	parser := New()
	parser.analysis(string(b))
	parser.dump()
}
