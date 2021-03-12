package configs

import (
	"github.com/legenove/cocore"
	"github.com/legenove/easyconfig/ifacer"
	"sync"
)

const (
	TrackerTypeRedis    = "redis"
	LogName             = "task"
	BootLoaderTypeQueue = "queue"
	BootLoaderTypeCron  = "cron"
	BootLoaderTypeAll   = "all"
	TaskClockLogName    = "task_clock"
	TaskConsumerLogName = "task_consumer"
	TaskProducerLogName = "task_producer"
)

type TasksConfigs struct {
	Tasks []TaskConfig `json:"tasks" mapstructure:"tasks"`
}

type TaskConfig struct {
	TaskName    string            `json:"task_name" mapstructure:"task_name"`
	Status      int               `json:"status" mapstructure:"status"`
	WorkerNum   int               `json:"worker_num" mapstructure:"worker_num"`
	ReceiverNum int               `json:"receiver_num" mapstructure:"receiver_num"`
	DeadTimeout int               `json:"dead_timeout" mapstructure:"dead_timeout"`
	RepeatTime  int               `json:"repeat_time" mapstructure:"repeat_time"`
	QueueType   string            `json:"queue_type" mapstructure:"queue_type"`
	QueueGroup  string            `json:"queue_group" mapstructure:"queue_group"`
	SendTimeout int               `json:"send_timeout" mapstructure:"send_timeout"`
	WarnTimeout int               `json:"warn_timeout" mapstructure:"warn_timeout"`
	TrackerRDIS *ReidsTrackerConf `json:"tracker_redis" mapstructure:"tracker_redis"`
}

func TaskConfigCompareEqual(c1, c2 *TaskConfig) bool {
	if c1.Status != c2.Status {
		return false
	}
	if c1.WorkerNum != c2.WorkerNum {
		return false
	}
	if c1.ReceiverNum != c2.ReceiverNum {
		return false
	}
	if c1.QueueType != c2.QueueType {
		return false
	}
	if c1.QueueGroup != c2.QueueGroup {
		return false
	}
	if c1.RepeatTime != c2.RepeatTime {
		return false
	}
	if c1.DeadTimeout != c2.DeadTimeout {
		return false
	}
	return true
}

type ReidsTrackerConf struct {
	ClusterName       string `json:"cluster_name" mapstructure:"cluster_name"`
	TaskKey           string `json:"task_key" mapstructure:"task_key"`
	DelayedTaskKey    string `json:"delayed_task_key" mapstructure:"delayed_task_key"`
	DeadLetterTaskKey string `json:"dead_letter_task_key" mapstructure:"dead_letter_task_key"`
	MaxReceiveCount   int    `json:"max_receive_count" mapstructure:"max_receive_count"`
	WarnTimeInterval  int    `json:"warn_time_tnterval" mapstructure:"warn_time_tnterval"`
	WarnQueueCount    int    `json:"warn_queue_count" mapstructure:"warn_queue_count"`
}

var ConsumerConf *ifacer.Configer
var lock sync.Mutex

func GetTaskConf() (*TasksConfigs, error) {
	if ConsumerConf != nil {
		return ConsumerConf.GetValue().(*TasksConfigs), nil
	}
	lock.Lock()
	defer lock.Unlock()
	if ConsumerConf == nil {
		c, err := cocore.Conf.Instance("consumer.yaml", "yaml", &TasksConfigs{}, nil, nil)
		if err != nil {
			return nil, err
		}
		ConsumerConf = c
	}
	return ConsumerConf.GetValue().(*TasksConfigs), nil
}
