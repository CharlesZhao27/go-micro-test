package discovery

import (
	"context"
	"fmt"
	"time"

	"your/service-discovery/internal/models"

	"github.com/google/uuid"
	clientv3 "go.etcd.io/etcd/client/v3"
)

const (
	serviceKeyPrefix = "/services/"
	defaultTTL       = 30 // seconds
)

type ServiceRegistry struct {
	client  *clientv3.Client
	leases  map[string]clientv3.LeaseID
	timeout time.Duration
}

func NewServiceRegistry(endpoints []string) (*ServiceRegistry, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %v", err)
	}

	return &ServiceRegistry{
		client:  client,
		leases:  make(map[string]clientv3.LeaseID),
		timeout: 5 * time.Second,
	}, nil
}

func (sr *ServiceRegistry) Register(ctx context.Context, req *models.RegisterRequest) (*models.ServiceInstance, error) {
	instance := &models.ServiceInstance{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Version:   req.Version,
		Host:      req.Host,
		Port:      req.Port,
		Metadata:  req.Metadata,
		Status:    "UP",
		UpdatedAt: time.Now(),
	}

	// Create lease
	lease, err := sr.client.Grant(ctx, defaultTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to create lease: %v", err)
	}

	// Store lease ID for future reference
	sr.leases[instance.ID] = lease.ID

	// Convert instance to JSON
	data, err := instance.ToJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal service data: %v", err)
	}

	// Create key in etcd with lease
	key := sr.serviceKey(instance.Name, instance.ID)
	_, err = sr.client.Put(ctx, key, string(data), clientv3.WithLease(lease.ID))
	if err != nil {
		return nil, fmt.Errorf("failed to register service: %v", err)
	}

	// Start keepalive
	go sr.keepAlive(ctx, instance.ID, lease.ID)

	return instance, nil
}

func (sr *ServiceRegistry) Deregister(ctx context.Context, serviceName, instanceID string) error {
	// Revoke lease if it exists
	if leaseID, exists := sr.leases[instanceID]; exists {
		_, err := sr.client.Revoke(ctx, leaseID)
		if err != nil {
			return fmt.Errorf("failed to revoke lease: %v", err)
		}
		delete(sr.leases, instanceID)
	}

	// Delete key
	key := sr.serviceKey(serviceName, instanceID)
	_, err := sr.client.Delete(ctx, key)
	return err
}

func (sr *ServiceRegistry) GetService(ctx context.Context, name string) ([]*models.ServiceInstance, error) {
	prefix := fmt.Sprintf("%s%s/", serviceKeyPrefix, name)
	resp, err := sr.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to get service instances: %v", err)
	}

	instances := make([]*models.ServiceInstance, 0)
	for _, kv := range resp.Kvs {
		instance, err := models.FromJSON(kv.Value)
		if err != nil {
			continue
		}
		instances = append(instances, instance)
	}

	if len(instances) == 0 {
		return nil, fmt.Errorf("no instances found for service: %s", name)
	}

	return instances, nil
}

func (sr *ServiceRegistry) GetAllServices(ctx context.Context) (map[string][]*models.ServiceInstance, error) {
	resp, err := sr.client.Get(ctx, serviceKeyPrefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to get all services: %v", err)
	}

	services := make(map[string][]*models.ServiceInstance)
	for _, kv := range resp.Kvs {
		instance, err := models.FromJSON(kv.Value)
		if err != nil {
			continue
		}

		if _, exists := services[instance.Name]; !exists {
			services[instance.Name] = make([]*models.ServiceInstance, 0)
		}
		services[instance.Name] = append(services[instance.Name], instance)
	}

	return services, nil
}

func (sr *ServiceRegistry) Watch(ctx context.Context, serviceName string) clientv3.WatchChan {
	prefix := fmt.Sprintf("%s%s/", serviceKeyPrefix, serviceName)
	return sr.client.Watch(ctx, prefix, clientv3.WithPrefix())
}

func (sr *ServiceRegistry) keepAlive(ctx context.Context, instanceID string, leaseID clientv3.LeaseID) {
	keepAliveChan, err := sr.client.KeepAlive(ctx, leaseID)
	if err != nil {
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case resp := <-keepAliveChan:
			if resp == nil {
				return
			}
		}
	}
}

func (sr *ServiceRegistry) serviceKey(serviceName, instanceID string) string {
	return fmt.Sprintf("%s%s/%s", serviceKeyPrefix, serviceName, instanceID)
}

func (sr *ServiceRegistry) Close() error {
	return sr.client.Close()
}
