package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	extinf = "#EXTINF:"
)

var (
	ErrNotFound   = errors.New("not found")
	ErrStatusCode = errors.New("http status code err")
)

type M3u8Parser struct {
	lineNum           int
	lastEnd           int64
	start             int64
	end               int64
	duration          int64
	totalDuration     int64
	totalDurationF    float64
	realTotalDuration int64
	firstTSStart      int64
	totalGap          int64
	maxGap            int64
	minGap            int64
	host              string
	scheme            string
	tsSavePath        string
	needDownload      bool
	parseJson         bool
	tsDurations       map[string]DurationInfo
}

type M []map[string]interface{}

func New(tsSavePath string, parseJson bool) *M3u8Parser {
	return &M3u8Parser{
		tsDurations:   map[string]DurationInfo{},
		parseJson:     parseJson,
		tsSavePath:    tsSavePath,
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

func (self *M3u8Parser) durationsToCsv() error {
	csv := "file, duration from ts, duration from file name, duration from extinf\n"
	for k, v := range self.tsDurations {
		csv += fmt.Sprintf("%s, %f, %d, %d\n", k, v.durationFromTs/90, v.durationFromFileName, v.durationFromExtinf)
	}
	file := fmt.Sprintf("%sts-durations.csv", self.tsSavePath)
	err := ioutil.WriteFile(file, []byte(csv), 0644)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

type DurationInfo struct {
	durationFromTs       float64
	durationFromFileName int64
	durationFromExtinf   int64
}

func (self *M3u8Parser) saveDurations(tsDuration float64) {
	durationInfo := DurationInfo{
		durationFromTs:       tsDuration,
		durationFromFileName: self.end - self.start,
		durationFromExtinf:   self.duration,
	}
	self.tsDurations[self.tsFile(self.start, self.end)] = durationInfo
}

func (self *M3u8Parser) parseFrameInfo(start, end int64) (float64, error) {
	tsfile := self.tsFile(start, end)
	jsonfile := self.jsonFile(start, end)
	b, err := ioutil.ReadFile(jsonfile)
	if err != nil {
		log.Println("read fail", jsonfile, err)
		return 0, err
	}
	m := make(map[string]M)
	if err := json.Unmarshal(b, &m); err != nil {
		log.Println(err)
		return 0, err
	}
	frames := m["frames"]
	firstFramePts := frames[0]["pkt_pts"].(float64)
	lastFramePts := frames[len(frames)-1]["pkt_pts"].(float64)
	tsDuration := lastFramePts - firstFramePts
	self.totalDurationF += tsDuration
	log.Println(tsfile, "parse done")
	return tsDuration, nil
}

func (self *M3u8Parser) tsFile(start, end int64) string {
	return fmt.Sprintf("%s%d-%d.ts", self.tsSavePath, start, end)
}

func (self *M3u8Parser) jsonFile(start, end int64) string {
	return fmt.Sprintf("%s%d-%d.json", self.tsSavePath, start, end)
}

func (self *M3u8Parser) dumpTSToJson(start, end int64) error {
	tsfile := self.tsFile(start, end)
	jsonFile := self.jsonFile(start, end)
	cmdstr := fmt.Sprintf("ffprobe -show_frames -of json %s > %s",
		tsfile, jsonFile)
	cmd := exec.Command("bash", "-c", cmdstr)
	_, err := cmd.CombinedOutput()
	if err != nil {
		log.Println("cmd:", cmdstr, "err:", err)
		return err
	}
	return nil
}

func (self *M3u8Parser) downloadTS(path string, start, end int64) error {
	log.Println("start to download", path)
	addr := self.scheme + "://" + self.host + path
	ts, err := httpGet(addr)
	if err != nil {
		return err
	}
	log.Println(path, "download done")
	file := fmt.Sprintf("%s%d-%d.ts", self.tsSavePath, start, end)
	err = ioutil.WriteFile(file, []byte(ts), 0644)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func (self *M3u8Parser) handleNewTS(line string) error {
	var err error
	self.start, self.end, err = self.getTimestamp(line)
	if err != nil {
		return err
	}
	if self.firstTSStart == 0 {
		self.firstTSStart = self.start
	}
	self.realTotalDuration = self.end - self.firstTSStart
	if self.needDownload {
		if err := self.downloadTS(line, self.start, self.end); err != nil {
			return err
		}
		if err := self.dumpTSToJson(self.start, self.end); err != nil {
			return err
		}
	}
	if self.parseJson {
		tsDuration, err := self.parseFrameInfo(self.start, self.end)
		if err != nil {
			return err
		}
		self.saveDurations(tsDuration)
	}
	return nil
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
		if err := self.handleNewTS(line); err != nil {
			return err
		}
	}
	self.check(line)

	return nil
}

func (self *M3u8Parser) analysis(m3u8 string) error {
	self.parseJson = true
	scanner := bufio.NewScanner(strings.NewReader(m3u8))
	for scanner.Scan() {
		if err := self.parseLine(scanner.Text()); err != nil {
			return err
		}
		self.lineNum++
	}
	return nil
}

func httpGet(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		log.Println(err)
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return "", err
	}
	if resp.StatusCode != 200 {
		log.Println("http read body err", resp.StatusCode)
		return "", ErrStatusCode
	}
	return string(body), nil
}

func (self *M3u8Parser) downloadAllTS(addr string) error {
	self.needDownload = true
	self.parseJson = true
	u, err := url.Parse(addr)
	if err != nil {
		log.Println(err)
		return err
	}
	self.host = u.Host
	self.scheme = u.Scheme
	m3u8, err := httpGet(addr)
	if err != nil {
		return err
	}
	file := fmt.Sprintf("%splayback.m3u8", self.tsSavePath)
	if err := ioutil.WriteFile(file, []byte(m3u8), 0644); err != nil {
		log.Println(err)
		return err
	}
	return self.analysis(m3u8)
}

func (self *M3u8Parser) dump() {
	fmt.Println("total duration from extinf:", self.totalDuration)
	fmt.Println("total duration from ts parse:", self.totalDurationF)
	fmt.Println("total duration from file name:", self.realTotalDuration)
	fmt.Println("gap duraion:", self.realTotalDuration-self.totalDuration)
	fmt.Println("total gap:", self.totalGap)
	fmt.Println("min gap:", self.minGap)
	fmt.Println("max gap:", self.maxGap)
}

func main() {
	log.SetFlags(log.Lshortfile)
	file := flag.String("file", "", "input .m3u8 file")
	url := flag.String("url", "", "hls playback url")
	// json文件已经存在
	parseJson := flag.Bool("parse-json", false, "hls playback url")
	tsSavePath := flag.String("ts-save-path", "./", "ts save path")
	flag.Parse()
	parser := New(*tsSavePath, *parseJson)
	if *file != "" {
		b, err := ioutil.ReadFile(*file)
		if err != nil {
			log.Println("open file", *file, "err", err)
			return
		}
		parser.analysis(string(b))
		parser.dump()
		parser.durationsToCsv()
	}
	if *url != "" {
		parser.downloadAllTS(*url)
		log.Println("all ts download done")
		parser.dump()
	}
	for {
		time.Sleep(3 * time.Second)
	}
}
