package redis

import (
    "context"
    "fmt"
    "github.com/go-redis/redis/v8"
    "github.com/legenove/turbo/configs"
    "github.com/legenove/turbo/message"
    "github.com/legenove/turbo/minware"
    "strconv"
    "time"
)

const (
    BLOCK_TIMEOUT  = 10 * time.Second
    MAX_DELAY_TIME = 60 * 15 // 最大延迟15分钟
)

type Tracker struct {
}

func NewTracker() *Tracker {
    return &Tracker{}
}

func (t *Tracker) EncodeType() string {
    return message.MESSAGE_ENCODE_TYPE_PROTO
}

func (t *Tracker) Sender(ctx context.Context, conf *configs.TaskConfig, msg []byte) error {
    client, err := minware.GetRedisClient(conf.TrackerRDIS.ClusterName)
    if err != nil {
        return err
    }
    _, err = client.LPush(ctx, conf.TrackerRDIS.TaskKey, msg).Result()
    if err != nil {
        return err
    }
    return nil
}

func (t *Tracker) SenderDelay(ctx context.Context, conf *configs.TaskConfig, msg []byte, delayTime int) error {
    if delayTime <= 0 || conf.TrackerRDIS.DelayedTaskKey == "" {
        return t.Sender(ctx, conf, msg)
    }
    if delayTime > MAX_DELAY_TIME {
        return fmt.Errorf("max delay time is %d min", MAX_DELAY_TIME/60)
    }
    now := time.Now().UTC().Unix()
    client, err := minware.GetRedisClient(conf.TrackerRDIS.ClusterName)
    if err != nil {
        return err
    }
    _, err = client.ZAdd(ctx, conf.TrackerRDIS.DelayedTaskKey,
        &redis.Z{Score: float64(now + int64(delayTime)), Member: msg}).Result()
    if err != nil {
        return err
    }
    return nil
}

func (t *Tracker) HasDelayReceiver(conf *configs.TaskConfig) bool {
    if conf.TrackerRDIS.DelayedTaskKey == "" {
        return false
    }
    return true
}

func (t *Tracker) DelayReceiver(conf *configs.TaskConfig,
    msgChan chan []byte, workPool chan struct{}, stop chan struct{}) error {
    if conf.TrackerRDIS.DelayedTaskKey == "" {
        return nil
    }
    client, err := minware.GetRedisClient(conf.TrackerRDIS.ClusterName)
    if err != nil {
        return err
    }
    isBreak := false
    go func() {
        select {
        case <-stop:
            isBreak = true
        }
    }()
    ctx := context.Background()
    fn := func(tx *redis.Tx) error {
        now := time.Now().UTC().Unix()
        opt := &redis.ZRangeBy{
            Min: "0", Max: strconv.FormatInt(now, 10),
            Offset: 0, Count: 1,
        }
        items, err := tx.ZRangeByScore(ctx, conf.TrackerRDIS.DelayedTaskKey, opt).Result()
        if err != nil && err != redis.Nil {
            return err
        }
        if len(items) != 1 {
            return redis.Nil
        }
        _, err = tx.Pipelined(ctx, func(pipe redis.Pipeliner) error {
            pipe.ZRem(ctx, conf.TrackerRDIS.DelayedTaskKey, items[0])
            return nil
        })
        if err != nil {
            return err
        }
        msgChan <- []byte(items[0])
        return nil
    }
    for {
        select {
        case <-workPool:
        }
    rwatch:
        if isBreak {
            // 返还worker
            workPool <- struct{}{}
            break
        }
        err := client.Watch(ctx, fn, conf.TrackerRDIS.DelayedTaskKey)
        if err != nil {
            if err == redis.Nil {
                time.Sleep(500 * time.Millisecond)
                goto rwatch
            } else {
                time.Sleep(500 * time.Millisecond)
                goto rwatch
            }
        } else {
            time.Sleep(200 * time.Microsecond)
        }
    }
    return nil
}

func (t *Tracker) Receiver(conf *configs.TaskConfig,
    msgChan chan []byte, workPool chan struct{}, stop chan struct{}) error {
    client, err := minware.GetRedisClient(conf.TrackerRDIS.ClusterName)
    if err != nil {
        return err
    }
    isBreak := false
    go func() {
        select {
        case <-stop:
            isBreak = true
        case <-time.After(time.Minute * 3):
            isBreak = true
        }
    }()
    for {
        select {
        case <-workPool:
        }
    brpop:
        if isBreak {
            // 返还worker
            workPool <- struct{}{}
            break
        }
        msg, err := client.BRPop(context.Background(), BLOCK_TIMEOUT, conf.TrackerRDIS.TaskKey).Result()
        if err != nil && err != redis.Nil {
            // 返还worker
            workPool <- struct{}{}
            time.Sleep(5 * time.Millisecond)
            return err
        }
        if len(msg) == 2 {
            msgChan <- []byte(msg[1])
            time.Sleep(1 * time.Millisecond)
        } else {
            time.Sleep(1 * time.Millisecond)
            goto brpop
        }
    }
    return nil
}
