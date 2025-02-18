package models

import (
	"encoding/json"
	"time"
)

type ServiceInstance struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Version   string            `json:"version"`
	Host      string            `json:"host"`
	Port      int               `json:"port"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Status    string            `json:"status"`
	UpdatedAt time.Time         `json:"updatedAt"`
}

type RegisterRequest struct {
	Name     string            `json:"name"`
	Version  string            `json:"version"`
	Host     string            `json:"host"`
	Port     int               `json:"port"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

func (s *ServiceInstance) ToJSON() ([]byte, error) {
	return json.Marshal(s)
}

func FromJSON(data []byte) (*ServiceInstance, error) {
	var service ServiceInstance
	err := json.Unmarshal(data, &service)
	return &service, err
}
