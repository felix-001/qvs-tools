package manager

type Handler func()

type Command struct {
	Handler            Handler
	Desc               string
	NeedIpParser       bool
	NeedRedis          bool
	NeedCK             bool
	NeedPrometheus     bool
	NeedNodeStreamInfo bool
	NeedNodeInfo       bool
}
