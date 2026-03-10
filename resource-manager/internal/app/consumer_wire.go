package app

import (
	"github.com/Meesho/BharatMLStack/resource-manager/internal/adapters/etcd"
	"github.com/Meesho/BharatMLStack/resource-manager/internal/adapters/redisq"
	"github.com/Meesho/BharatMLStack/resource-manager/internal/application"
	"github.com/Meesho/BharatMLStack/resource-manager/pkg/config"
)

func BuildConsumerService() (*application.ConsumerService, error) {
	envCfg := config.Instance()
	queueAdapter := redisq.NewInMemoryQueueAdapter()

	watchManager := application.NewMockWatchManager()
	callbacks := application.NewMockCallbackDispatcher()

	if envCfg.UseMockAdapters {
		return application.NewConsumerService(
			envCfg.ConsumerID,
			envCfg.ConsumerGroupID,
			envCfg.QueuePartitionCount,
			queueAdapter,
			etcd.NewMemoryQueueLeaseStore(),
			etcd.NewMemoryConsumerMembershipStore(),
			watchManager,
			callbacks,
			envCfg.QueuePollInterval,
			envCfg.QueueLeaseRenew,
			envCfg.RebalanceDrainTimeout,
		), nil
	}

	etcdClient, err := etcd.NewClient(etcd.ClientConfig{
		Endpoints: envCfg.EtcdEndpoints,
		Username:  envCfg.EtcdUsername,
		Password:  envCfg.EtcdPassword,
		Timeout:   envCfg.EtcdTimeout,
	})
	if err != nil {
		return nil, err
	}
	return application.NewConsumerService(
		envCfg.ConsumerID,
		envCfg.ConsumerGroupID,
		envCfg.QueuePartitionCount,
		queueAdapter,
		etcd.NewEtcdQueueLeaseStore(etcdClient.Raw(), envCfg.QueueLeaseTTL),
		etcd.NewEtcdConsumerMembershipStore(etcdClient.Raw(), envCfg.ConsumerMembershipTTL),
		watchManager,
		callbacks,
		envCfg.QueuePollInterval,
		envCfg.QueueLeaseRenew,
		envCfg.RebalanceDrainTimeout,
	), nil
}
