package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

func getVolcLiveSourceFlag(sk string, t time.Time) (string, string) {
	date := t.Format(time.RFC3339)
	h := hmac.New(sha256.New, []byte(sk))
	h.Write([]byte(date))
	result := h.Sum(nil)
	return hex.EncodeToString(result), date
}

// go run . -cmd stag -subcmd volc -domain huyap2p-source.bytefcdn.com -bkt huyalive -stream 78941969-2559461593-10992803837303062528-2693342886-10057-A-0-1-imgplus_540_2_66
func (s *Parser) fetchVolcOriginUrl() {
	addr := fmt.Sprintf("http://%s/%s/%s.slice?baseIndex=0&quickTime=10000&cdn=qiniuyun",
		s.conf.Domain, s.conf.Bucket, s.conf.Stream)
	s.logger.Info().Str("sk", s.conf.Sk).Msg("fetchVolcOriginUrl")
	flag, date := getVolcLiveSourceFlag(s.conf.Sk, time.Now())
	s.logger.Info().Str("flag", flag).Str("date", date).Msg("fetchVolcOriginUrl")
	headers := map[string]string{
		"Date":                  date,
		"volc-live-source-flag": flag,
		"Host":                  s.conf.Domain,
	}
	resp, err := httpReq("GET", addr, "", headers)
	if err != nil {
		s.logger.Error().Err(err).Msg("fetchVolcOriginUrl")
		return
	}
	fmt.Println(resp)
}
