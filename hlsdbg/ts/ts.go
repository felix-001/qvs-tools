package ts

type TsMgr struct {
}

func New() *TsMgr {
	return &TsMgr{}
}

func (self *TsMgr) Fetch(addr string) error {
	return nil
}
