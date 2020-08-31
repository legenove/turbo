package tasks

import (
	"github.com/legenove/cocore"
	"go.uber.org/zap"
	"go_svc/ocls_tasks/tasks/configs"
	"go_svc/ocls_tasks/tasks/message"
	"time"
)

const (
	LOG_TYPE_TASK            = "task"
	LOG_EVENT_PRODUCER       = "producer"
	LOG_EVENT_CONSUMER       = "consumer"
	LOG_EVENT_RECEIVER_START = "receiver_start"
	LOG_EVENT_RECEIVER_END   = "receiver_end"
	LOG_EVENT_RECEIVER       = "receiver"
)

var Logger TurboLogger

func GetLogger() TurboLogger {
	if Logger == nil {
		Logger = &DefaultLogger{"turbo_task"}
	}
	return Logger
}

func RegisterLogger(log TurboLogger) {
	Logger = log
}

type LoggerPrinter struct {
	logEvent string
	taskMsg  *message.Message
	taskConf *configs.TaskConfig
	err      error
	msg      string
	delay    int
	Duration time.Duration
}

func (p *LoggerPrinter) GetTaskField() []zap.Field {
	var fields = make([]zap.Field, 0, 16)
	fields = append(fields,
		zap.String("event", p.logEvent),
		zap.Namespace("properties"),
		zap.String("uuid", p.taskMsg.Uuid),
		zap.String("task_name", p.taskMsg.TaskName),
		zap.Uint32("receive_count", p.taskMsg.ReceiveCount),
		zap.Int64("send_time", p.taskMsg.SendTime),
		zap.Duration("duration", p.Duration),
	)
	if p.err != nil {
		fields = append(fields,
			zap.ByteString("task_msg", p.taskMsg.GetArgs()),
		)
	}
	if p.delay != 0 {
		fields = append(fields,
			zap.Int("delay", p.delay),
		)
	}
	return fields
}

func (p *LoggerPrinter) GetTaskConfField() []zap.Field {
	var fields = make([]zap.Field, 0, 16)
	fields = append(fields,
		zap.String("event", p.logEvent),
		zap.Namespace("properties"),
		zap.String("task_name", p.taskConf.TaskName),
		zap.Int("worker_num", p.taskConf.WorkerNum),
		zap.Int("receive_num", p.taskConf.ReceiverNum),
		zap.String("queue_group", p.taskConf.QueueGroup),
		zap.String("queue_type", p.taskConf.QueueType),
	)
	return fields
}

type TurboLogger interface {
	Print(*LoggerPrinter)
}

type DefaultLogger struct {
	logName string
}

func (logger *DefaultLogger) Logger() (*zap.Logger, error) {
	l, err := cocore.LogPool.Instance(logger.logName)
	if err != nil {
		return nil, err
	}
	l = l.With(zap.String("log_type", LOG_TYPE_TASK))
	return l, nil
}

func (logger *DefaultLogger) Print(p *LoggerPrinter) {
	l, err := logger.Logger()
	if err != nil {
		return
	}
	switch p.logEvent {
	case LOG_EVENT_PRODUCER, LOG_EVENT_CONSUMER:
		if p.err != nil {
			l.Error(p.err.Error(), p.GetTaskField()...)
		} else {
			l.Info(p.msg, p.GetTaskField()...)
		}
	case LOG_EVENT_RECEIVER_START, LOG_EVENT_RECEIVER, LOG_EVENT_RECEIVER_END:
		if p.err != nil {
			l.Error(p.err.Error(), p.GetTaskConfField()...)
		} else {
			l.Info(p.msg, p.GetTaskConfField()...)
		}
	}
}
