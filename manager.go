package turbo

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/legenove/turbo/configs"
)

type ServerManager struct {
	sync.Mutex
	wg          sync.WaitGroup
	server      *TaskServer
	Error       error
	QueueGroup  string
	stop        chan struct{}
	startServer bool
}

var Manager *ServerManager
var mgrLock sync.Mutex

func GetManager() *ServerManager {
	mgrLock.Lock()
	defer mgrLock.Unlock()
	if Manager == nil {
		Manager = newServerManager()
	}
	return Manager
}

func (m *ServerManager) RegisterWorker(name string, worker WorkerIface) error {
	ser := m.GetServer()
	if ser == nil {
		return m.Error
	}
	return ser.RegisterWorker(name, worker)
}

func (m *ServerManager) GetServer() *TaskServer {
	m.Lock()
	defer m.Unlock()
	if m.server == nil {
		m.stop = make(chan struct{})
		m.server, m.Error = NewTaskServer(m.stop)
	}
	return m.server
}

func (m *ServerManager) GetStopChan() chan struct{} {
	m.Lock()
	defer m.Unlock()
	return m.stop
}

func newServerManager() *ServerManager {
	return &ServerManager{wg: sync.WaitGroup{}}
}

func listenSignal() {
	stopSig := make(chan os.Signal, 1)
	signal.Notify(stopSig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case <-stopSig:
			Manager.CloseStop()
		}
	}()
}

func listenFileChange() {
	go func() {
		for {
			select {
			case <-configs.ConsumerConf.OnChange:
				fmt.Println("ConsumerConf.OnChange")
				GetManager().ReLoadConf()
			}
		}
	}()
}

func (m *ServerManager) CloseStop() {
	m.Lock()
	defer m.Unlock()
	close(m.stop)
}

func (m *ServerManager) ReLoadConf() {
	hsvc := Manager.GetServer()
	newCfg, _ := configs.GetTaskConf()
	restart := false
	for _, t := range newCfg.Tasks {
		ot, err := hsvc.GetTask(t.TaskName)
		if err != nil || ot.Worker == nil {
			continue
		}
		if !configs.TaskConfigCompareEqual(&t, ot.Conf) {
			restart = true
			break
		}
	}
	fmt.Println("restart", restart)
	if !restart {
		return
	}
	m.RestartServer(hsvc, newCfg)
}

func (m *ServerManager) RestartServer(hsvc *TaskServer, newCfg *configs.TasksConfigs) {
	m.Lock()
	defer m.Unlock()
	if m.startServer {
		m.wg.Add(1)
		close(m.stop)
		m.stop = make(chan struct{})
		m.server, m.Error = hsvc.NewServer(newCfg, m.stop)
		if m.Error != nil {
			m.wg.Done()
			return
		}
		go func() {
			m.server.Start(m.QueueGroup)
			fmt.Println("stop server")
			m.wg.Done()
		}()
	} else {
		// 只更新server
		m.server, m.Error = hsvc.NewServer(newCfg, m.stop)
	}
}

func (m *ServerManager) Run(queueGroup string) error {
	m.QueueGroup = queueGroup
	svc := m.GetServer()
	if svc == nil {
		return m.Error
	}
	m.startServer = true
	listenSignal()
	listenFileChange()
	m.wg.Add(1)
	go func() {
		svc.Start(queueGroup)
		fmt.Println("stop server")
		m.wg.Done()
	}()
	m.wg.Wait()
	return m.Error
}
