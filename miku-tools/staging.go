package main

// 临时的代码放到这里

func (s *Parser) Staging() {
	switch s.conf.SubCmd {
	case "getpcdn":
		s.getPcdnFromSchedAPI(true, false)
	case "volc":
		s.fetchVolcOriginUrl()
	}
}
