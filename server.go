package turbo

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/legenove/turbo/configs"
	"github.com/legenove/turbo/message"
	"github.com/legenove/turbo/trackers"
)

type TaskServer struct {
	Mutex sync.RWMutex
	Conf  *configs.TasksConfigs
	Tasks map[string]*Task
	stop  chan struct{}
}

type WorkerIface interface {
	Process(ctx *message.MsgContext, msg []byte) error // 执行方法
}

func NewTaskServer(stop chan struct{}) (*TaskServer, error) {
	cfg, err := configs.GetTaskConf()
	if err != nil {
		return nil, fmt.Errorf("Invalid task conf; err:%s\n", err)
	}
	// TODO == 0 时 再斟酌如何处理
	if len(cfg.Tasks) == 0 {
		return nil, fmt.Errorf("Empty task conf")
	}

	srv := &TaskServer{Conf: cfg, Tasks: make(map[string]*Task), stop: stop}
	for _, config := range cfg.Tasks {
		_conf := config
		tracker, err := trackers.GetTracker(&_conf)
		if err != nil {
			return nil, fmt.Errorf("Invalid %s task config Empty task conf;%v\n", config.TaskName, config)
		}
		srv.Tasks[_conf.TaskName] = NewTask(tracker, &_conf)
	}
	return srv, nil
}

func (srv *TaskServer) NewServer(cfg *configs.TasksConfigs, stop chan struct{}) (*TaskServer, error) {
	nsrv := &TaskServer{Conf: cfg, Tasks: make(map[string]*Task), stop: stop}
	for _, config := range cfg.Tasks {
		_conf := config
		tracker, err := trackers.GetTracker(&_conf)
		if err != nil {
			return nil, fmt.Errorf("Invalid %s task config Empty task conf;%v\n", config.TaskName, config)
		}
		task := NewTask(tracker, &_conf)
		_t, _ := srv.GetTask(_conf.TaskName)
		if _t != nil {
			task.Worker = _t.Worker
		}
		nsrv.Tasks[_conf.TaskName] = task
	}
	return nsrv, nil
}

func (srv *TaskServer) RegisterWorker(name string, worker WorkerIface) error {
	srv.Mutex.Lock()
	defer srv.Mutex.Unlock()

	tsk, ok := srv.Tasks[name]
	if !ok {
		return fmt.Errorf("Task %s conf does not exist", name)
	}

	if tsk.Worker != nil {
		return fmt.Errorf("Task %s already exist", name)
	}

	tsk.Worker = worker
	return nil
}

func (srv *TaskServer) SendMsg(taskName string, ctx context.Context,
	msgCtx map[string]string, msg []byte, delayTime ...int) (string, error) {
	srv.Mutex.Lock()
	defer srv.Mutex.Unlock()

	tsk, ok := srv.Tasks[taskName]
	if !ok {
		return "", fmt.Errorf("Task %s conf does not exist", taskName)
	}
	return tsk.Publish(ctx, msg, msgCtx, delayTime...)
}

func (srv *TaskServer) GetTask(name string) (*Task, error) {
	srv.Mutex.RLock()
	defer srv.Mutex.RUnlock()
	tsk, ok := srv.Tasks[name]
	if !ok {
		return nil, fmt.Errorf("Task not registered error: %s", name)
	}
	return tsk, nil
}

func (srv *TaskServer) Start(queueGroup string) error {
	srv.Mutex.RLock()
	if len(srv.Tasks) == 0 {
		srv.Mutex.RUnlock()
		return fmt.Errorf("No Task registered")
	}
	var ks = make([]string, 0, len(srv.Tasks))
	for _, task := range srv.Tasks {
		s := task.Conf.TaskName
		ks = append(ks, s)
	}
	srv.Mutex.RUnlock()
	var count int64 = 0
	wg := sync.WaitGroup{}
	for _, k := range ks {
		t, _ := srv.GetTask(k)
		if t != nil && t.Worker != nil {
			wg.Add(1)
			go func(task *Task) {
				defer wg.Done()
				task.Consumer(srv, queueGroup, &count, srv.stop)
			}(t)
		}
	}
	time.Sleep(1 * time.Second)

	wg.Wait()
	return nil
}
