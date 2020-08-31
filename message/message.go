package message

import (
	"sync"
	"time"
)


const (
	MESSAGE_ENCODE_TYPE_PROTO = "proto"
	MESSAGE_ENCODE_TYPE_JSON = "json"
)

type MsgContext struct {
	Uuid         string
	SendTime     int64
	TaskName     string
	ReceiveCount uint32
	ProcessTime  int64
	Context      map[string]string
}

var MsgContextPool = sync.Pool{
	New: func() interface{} {
		return &MsgContext{}
	},
}

func NewMsgContext(message *Message) *MsgContext {
	o := MsgContextPool.Get().(*MsgContext)
	o.Uuid = message.GetUuid()
	o.SendTime = message.GetSendTime()
	o.TaskName = message.GetTaskName()
	o.ReceiveCount = message.GetReceiveCount()
	o.ProcessTime = time.Now().UnixNano() / 1e6
	o.Context = message.GetContext()
	return o
}

func NewMessage(uuid, taskName string) *Message {
	return &Message{
		Uuid:         uuid,
		TaskName:     taskName,
		SendTime:     time.Now().UnixNano() / 1e6,
		ReceiveCount: 0,
	}
}

var MsgPool = sync.Pool{
	New: func() interface{} {
		return &Message{}
	},
}

func GetMessageFromPool() *Message {
	msg := MsgPool.Get().(*Message)
	msg.Reset()
	return msg
}

func (m *Message) Put() {
	MsgPool.Put(m)
}

func (m *Message) Clear() {
	m.Uuid = ""
	m.SendTime = 0
	m.Args = nil
	m.TaskName = ""
	m.ReceiveCount = 0
	m.Context = nil
}

func (m *Message) AddContext(key, value string) {
	if m.GetContext() == nil {
		m.Context = map[string]string{key: value}
	} else {
		m.Context[key] = value
	}
}

func (m *Message) SetContextByMap(kv map[string]string) {
	if kv != nil {
		for k, v := range kv {
			m.AddContext(k, v)
		}
	}
}

func (m *Message) SetArgs(args []byte) {
	m.Args = args
}
