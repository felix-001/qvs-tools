package callback

import "github.com/qbox/mikud-live/common/model"

type Callback interface {
	GetNodeAllStreams(nodeId string) (*model.NodeStreamInfo, error)
}
