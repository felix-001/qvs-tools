package util

import (
	"regexp"

	"github.com/qiniu/x/log"
)

func Re(pattern, replace, raw string) {
	reg, err := regexp.Compile(pattern)
	if err != nil {
		log.Println("compile rule err, pattern:", pattern, "err:", err)
		return
	}

	if reg.MatchString(raw) {
		res := reg.ReplaceAllString(raw, replace)
		log.Println("raw:", raw, "pattern:", pattern, "replace:", replace, "res:", res)
	} else {
		log.Println("raw not match pattern", raw, pattern, replace)
	}
}
