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
	matches := reg.FindAllStringSubmatch(raw, -1)
	if len(matches) > 0 {
		for i, match := range matches {
			log.Printf("Match %d:\n", i+1)
			for j, group := range match {
				log.Printf("  Group %d: %s\n", j, group)
			}
		}
	}

	if reg.MatchString(raw) {
		res := reg.ReplaceAllString(raw, replace)
		log.Println("raw:", raw, "pattern:", pattern, "replace:", replace, "res:", res)
	} else {
		log.Println("raw not match pattern", raw, pattern, replace)
	}
}
