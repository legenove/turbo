package tasks

import (
	"context"
	"errors"
	"github.com/golang/protobuf/proto"
	jsoniter "github.com/json-iterator/go"
	"github.com/legenove/random"
	"go_svc/ocls_tasks/tasks/configs"
	"go_svc/ocls_tasks/tasks/message"
	"go_svc/ocls_tasks/tasks/trackers"
	"sync"
	"sync/atomic"
	"time"
)

const (
	MaxReceiverNumber = 100
	MaxWorkerNumber   = 1000
)

type Task struct {
	Conf    *configs.TaskConfig
	Tracker trackers.TrackerIface // TODO Trackerface
	Worker  WorkerIface
	stop    chan struct{}
}

func NewTask(tracker trackers.TrackerIface, config *configs.TaskConfig) *Task {
	return &Task{Tracker: tracker, Conf: config}
}

func (t *Task) Publish(ctx context.Context, msg []byte, msgCtx map[string]string, delayTime ...int) (string, error) {
	uuid := random.UuidV5()
	m := message.NewMessage(uuid, t.Conf.TaskName)
	m.SetContextByMap(msgCtx)
	m.SetArgs(msg)
	start := time.Now()
	var bm []byte
	var err error
	printer := &LoggerPrinter{taskMsg: m, logEvent: LOG_EVENT_PRODUCER, msg: "publish msg"}
	defer func() {
		if err != nil {
			printer.err = err
		}
		printer.Duration = time.Now().Sub(start)
		GetLogger().Print(printer)
	}()
	if t.Tracker.EncodeType() == message.MESSAGE_ENCODE_TYPE_PROTO {
		bm, err = proto.Marshal(m)
	} else if t.Tracker.EncodeType() == message.MESSAGE_ENCODE_TYPE_JSON {
		bm, err = jsoniter.Marshal(m)
	} else {
		return uuid, errors.New("Message Support EncodeType: " + t.Tracker.EncodeType())
	}
	if err != nil {
		return uuid, err
	}
	if len(delayTime) > 0 && delayTime[0] > 0{
		err = t.Tracker.SenderDelay(ctx, t.Conf, bm, delayTime[0])
		printer.delay = delayTime[0]
	} else {
		err = t.Tracker.Sender(ctx, t.Conf, bm)
	}
	return uuid, err
}

func (t *Task) Consumer(svc *TaskServer, queueGroup string, count *int64, stop chan struct{}) {
	if queueGroup != "*" && t.Conf.QueueGroup != queueGroup {
		return
	}
	if t.Conf.Status == 0 {
		return
	}
	_printer := &LoggerPrinter{taskConf: t.Conf, logEvent: LOG_EVENT_RECEIVER_START}
	GetLogger().Print(_printer)
	defer func() {
		_printer.logEvent = LOG_EVENT_RECEIVER_END
		GetLogger().Print(_printer)
	}()
	atomic.AddInt64(count, 1)
	t.stop = make(chan struct{})
	receiverNum := t.Conf.ReceiverNum
	if receiverNum > MaxReceiverNumber {
		receiverNum = MaxReceiverNumber
	}
	workerNum := t.Conf.WorkerNum
	if workerNum > MaxWorkerNumber {
		workerNum = MaxWorkerNumber
	} else if workerNum < receiverNum + 1 {
		workerNum = receiverNum + 1
	}
	workPool := make(chan struct{}, MaxWorkerNumber+1)
	msgChan := make(chan []byte, MaxWorkerNumber)
	receiverPool := make(chan struct{}, MaxReceiverNumber)
	delayReceiverPool := make(chan struct{}, MaxReceiverNumber)
	for i := 0; i <= workerNum; i++ {
		workPool <- struct{}{}
	}
	for i := 0; i < receiverNum; i++ {
		receiverPool <- struct{}{}
	}
	// 最多3个
	for i := 0; i < receiverNum/40+1; i++ {
		delayReceiverPool <- struct{}{}
	}
	wkWG := sync.WaitGroup{}
	recvWG := sync.WaitGroup{}
	go func() {
		recvWG.Add(1)
		defer recvWG.Done()
		if !t.Tracker.HasDelayReceiver(t.Conf) {
			return
		}
		//  delay receiver
		for {
			isBreak := false
			select {
			case <-delayReceiverPool:
				recvWG.Add(1)
				go func() {
					defer recvWG.Done()
					err := t.Tracker.DelayReceiver(t.Conf, msgChan, workPool, t.stop)
					if err != nil {
						printer := &LoggerPrinter{taskConf: t.Conf, logEvent: LOG_EVENT_RECEIVER}
						printer.err = err
						GetLogger().Print(printer)
					}
					delayReceiverPool <- struct{}{}
					// 短暂停顿
					time.Sleep(1000 * time.Millisecond)
				}()
			case <-stop:
				isBreak = true
				break
			}
			if isBreak {
				break
			}
		}
	}()
	go func() {
		recvWG.Add(1)
		defer recvWG.Done()
		// receiver
		for {
			isBreak := false
			select {
			case <-receiverPool:
				recvWG.Add(1)
				go func() {
					defer recvWG.Done()
					err := t.Tracker.Receiver(t.Conf, msgChan, workPool, t.stop)
					if err != nil {
						printer := &LoggerPrinter{taskConf: t.Conf, logEvent: LOG_EVENT_RECEIVER}
						printer.err = err
						GetLogger().Print(printer)
					}
					// 执行一段时间则退出重来
					receiverPool <- struct{}{}
					// 短暂停顿
					time.Sleep(1 * time.Millisecond)
				}()
			case <-stop:
				isBreak = true
				break
			}
			if isBreak {
				break
			}
		}
	}()
	go func() {
		wkWG.Add(1)
		defer wkWG.Done()
		// consumer
		for {
			isBreak := false
			select {
			case b, ok := <-msgChan:
				if !ok {
					isBreak = true
					break
				}
				wkWG.Add(1)
				go func() {
					var _t *Task
					var err error
					defer func() {
						workPool <- struct{}{}
						wkWG.Done()
					}()
					msg := message.GetMessageFromPool()
					if t.Tracker.EncodeType() == message.MESSAGE_ENCODE_TYPE_PROTO {
						err = proto.Unmarshal(b, msg)
					} else if t.Tracker.EncodeType() == message.MESSAGE_ENCODE_TYPE_JSON {
						err = jsoniter.Unmarshal(b, &msg)
					} else {
						return
					}
					if err != nil || msg == nil {
						return
					}
					msg.ReceiveCount += 1
					ctx := message.NewMsgContext(msg)
					_t, err = svc.GetTask(msg.TaskName)
					if err != nil {
						printer := &LoggerPrinter{taskMsg: msg, logEvent: LOG_EVENT_CONSUMER}
						printer.err = errors.New("task conf not found")
						GetLogger().Print(printer)
						return
					}
					start := time.Now()
					err = _t.Worker.Process(ctx, msg.GetArgs())
					printer := &LoggerPrinter{taskMsg: msg, logEvent: LOG_EVENT_CONSUMER}
					printer.Duration = time.Now().Sub(start)
					if err != nil {
						printer.err = err
						GetLogger().Print(printer)
						// 失败重新发送处理
						if int(msg.ReceiveCount) < _t.Conf.RepeatTime {
							if t.Tracker.EncodeType() == message.MESSAGE_ENCODE_TYPE_PROTO {
								b, err = proto.Marshal(msg)
							} else if t.Tracker.EncodeType() == message.MESSAGE_ENCODE_TYPE_JSON {
								b, err = jsoniter.Marshal(msg)
							} else {
								return
							}
							if err != nil {
								return
							}
							ctx := context.Background()
							err = _t.Tracker.Sender(ctx, _t.Conf, b)
							if err != nil {
								return
							}
						}
						return
					}
					printer.err = nil
					GetLogger().Print(printer)
				}()
			}
			if isBreak {
				break
			}
		}
	}()
	for {
		isBreak := false
		select {
		case <-stop:
			close(t.stop)
			isBreak = true
		}
		if isBreak {
			break
		}
	}
	recvWG.Wait()
	close(msgChan)
	wkWG.Wait()
}
