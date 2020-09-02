package minware

import (
	"github.com/go-redis/redis"
	"github.com/legenove/redis_client"
)

var redisClientMap map[string]*redis.Client
var redisClusterClientMap map[string]*redis.ClusterClient

func RegisterRedisClient(clusterName string, client *redis.Client) {
	if redisClientMap == nil {
		redisClientMap = make(map[string]*redis.Client)
	}
	redisClientMap[clusterName] = client
}

func RegisterRedisClusterClient(clusterName string, client *redis.ClusterClient) {
	if redisClientMap == nil {
		redisClusterClientMap = make(map[string]*redis.ClusterClient)
	}
	redisClusterClientMap[clusterName] = client
}

func GetRedisClient(name string) (*redis.Client, error) {
	if redisClientMap != nil {
		if client, ok := redisClientMap[name]; ok && client != nil {
			return client, nil
		}
	}
	return redis_client.GetRedisClient(name)
}

func GetRedisClusterClient(clusterName string) (*redis.ClusterClient, error) {
	if redisClientMap != nil {
		if client, ok := redisClusterClientMap[clusterName]; ok && client != nil {
			return client, nil
		}
	}
	return redis_client.GetRedisCluster(clusterName)
}
