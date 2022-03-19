package register

import (
	"context"
	"encoding/json"
	"fmt"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type Reg struct {
	Id        string `json:"id"`
	Namespace string `json:"namespace"`
	Version   string `json:"version"`
	Ref       string `json:"ref"`
}
type Service struct {
	Id        string
	client    *RegisterClient
	lease     clientv3.Lease
	leaseId   clientv3.LeaseID
	Namespace string `mapstructure:"namespace"`
	Version   string `mapstructure:"version"`
	Ref       string `mapstructure:"ref"`
}

func (s *Service) Init(client *RegisterClient) error {
	s.client = client
	s.lease = clientv3.NewLease(client.Client)
	resp, err := s.lease.Grant(context.TODO(), 5)
	if err != nil {
		return err
	}
	s.leaseId = resp.ID
	_, err = s.lease.KeepAlive(context.TODO(), s.leaseId)
	return nil
}

func (s *Service) Register() error {
	kv := clientv3.NewKV(s.client.Client)
	reg := Reg{
		Id:        s.Id,
		Namespace: s.Namespace,
		Version:   s.Version,
		Ref:       s.Ref,
	}
	rawData, err := json.Marshal(reg)
	if err != nil {
		return err
	}
	_, err = kv.Put(context.TODO(), fmt.Sprintf("/reg/%s", s.Id), string(rawData), clientv3.WithLease(s.leaseId))
	if err != nil {
		return err
	}
	return err
}

func (s *Service) UnRegister() error {
	kv := clientv3.NewKV(s.client.Client)
	_, err := kv.Delete(context.TODO(), fmt.Sprintf("/reg/%s", s.Id))
	if err != nil {
		return err
	}
	return nil
}
