package nacos

import (
	"fmt"

	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

// GetServiceInstance 获取指定服务的一个健康实例
func (p *NacosPlugin) GetServiceInstance(serviceName string, group string) (*model.Instance, error) {
	if p.namingClient == nil {
		return nil, fmt.Errorf("nacos client not initialized")
	}

	if group == "" {
		group = "DEFAULT_GROUP"
	}

	instance, err := p.namingClient.SelectOneHealthyInstance(vo.SelectOneHealthInstanceParam{
		ServiceName: serviceName,
		GroupName:   group,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get service instance: %v", err)
	}

	return instance, nil
}

// GetAllServiceInstances 获取指定服务的所有实例
func (p *NacosPlugin) GetAllServiceInstances(serviceName string, group string) ([]model.Instance, error) {
	if p.namingClient == nil {
		return nil, fmt.Errorf("nacos client not initialized")
	}

	if group == "" {
		group = "DEFAULT_GROUP"
	}

	instances, err := p.namingClient.SelectAllInstances(vo.SelectAllInstancesParam{
		ServiceName: serviceName,
		GroupName:   group,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get service instances: %v", err)
	}

	return instances, nil
}

// GetHealthyServiceInstances 获取指定服务的所有健康实例
func (p *NacosPlugin) GetHealthyServiceInstances(serviceName string, group string) ([]model.Instance, error) {
	if p.namingClient == nil {
		return nil, fmt.Errorf("nacos client not initialized")
	}

	if group == "" {
		group = "DEFAULT_GROUP"
	}

	instances, err := p.namingClient.SelectInstances(vo.SelectInstancesParam{
		ServiceName: serviceName,
		GroupName:   group,
		HealthyOnly: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get healthy service instances: %v", err)
	}

	return instances, nil
}
