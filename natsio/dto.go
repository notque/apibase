package natsio
import (
	"time"
)

type RequestType string

const (
	Publish RequestType = "PUB"
	PublishRequest RequestType = "PUBREQ"
	Request RequestType = "REQ"
)

type Trail struct {
	AppName string
	PutType RequestType
	Time time.Time
}

type NatsContext struct {
	AppTrail []Trail
	TraceID string
}

func (n *NatsContext) appendTrail(appName string, requestType RequestType){
	n.AppTrail = append(n.AppTrail, Trail{appName, requestType, time.Now()})
}

type NatsDTO struct {
	NatsCtx NatsContext
	Error error
}

func (n *NatsDTO) Context() *NatsContext {
	return &n.NatsCtx
}

func (n *NatsDTO) NewContext(nC *NatsContext) {
	n.NatsCtx = *nC
}