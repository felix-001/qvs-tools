package manager

type Handler func()

type Command struct {
	Handler   Handler
	Desc      string
	Resources []string
}
