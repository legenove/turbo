package trackers

import (
	"context"
	"go_svc/ocls_tasks/tasks/configs"
	"go_svc/ocls_tasks/tasks/trackers/redis"
)

const (
	QUEUE_TYPE_REDIS = "redis"
)

type TrackerIface interface {
	Sender(ctx context.Context, conf *configs.TaskConfig, msg []byte) error
	SenderDelay(ctx context.Context, conf *configs.TaskConfig, msg []byte, delayTime int) error
	Receiver(conf *configs.TaskConfig, msgChan chan []byte,workPool chan struct{},stop chan struct{}) error
	DelayReceiver(conf *configs.TaskConfig, msgChan chan []byte,workPool chan struct{},stop chan struct{}) error
	HasDelayReceiver(conf *configs.TaskConfig) bool
	EncodeType() string
}

func GetTracker(queueType *configs.TaskConfig) (TrackerIface, error) {
	if queueType.QueueType == QUEUE_TYPE_REDIS {
		return redis.NewTracker(), nil
	}
	return nil, nil
}