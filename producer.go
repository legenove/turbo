package turbo

import (
	"context"
	"sync"
)

var ProducerServer *TaskServer
var producerMutex sync.Mutex

func GetProducerServer() (*TaskServer, error) {
	producerMutex.Lock()
	defer producerMutex.Unlock()
	var err error
	if ProducerServer == nil {
		ProducerServer, err = NewTaskServer(nil)
	}
	return ProducerServer, err
}

type Producer struct {
	ctx      context.Context
	taskName string
	msgCtx   map[string]string
}

func NewProducer(taskName string) *Producer {
	return &Producer{
		taskName: taskName,
	}
}

func (p *Producer) New() *Producer {
	o := &Producer{}
	o.taskName = p.taskName
	return o
}

func (p *Producer) SetMsgCtx(msgCtx map[string]string) *Producer {
	p.msgCtx = msgCtx
	return p
}

func (p *Producer) SetCtx(ctx context.Context) *Producer {
	p.ctx = ctx
	return p
}

func (p *Producer) Publish(msg []byte, delayTime ...int) (string, error) {
	svc, err := GetProducerServer()
	if svc == nil {
		return "", err
	}
	return svc.SendMsg(p.taskName, p.ctx, p.msgCtx, msg, delayTime...)
}
