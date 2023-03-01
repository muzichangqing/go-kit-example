package main

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/hashicorp/consul/api"
)

type ConsulServiceRegistry struct {
	client         api.Client
	localServiceID string
}

func (c ConsulServiceRegistry) Register(id, name, host string, port int, isSecure bool, tags []string) error {
	registration := new(api.AgentServiceRegistration)
	registration.ID = id
	registration.Name = name
	registration.Address = host
	registration.Port = port
	if isSecure {
		tags = append(tags, "secure=true")
	} else {
		tags = append(tags, "secure=false")
	}
	registration.Tags = tags

	check := new(api.AgentServiceCheck)
	schema := "http"
	if isSecure {
		schema = "https"
	}
	check.HTTP = fmt.Sprintf("%s://%s:%d/health", schema, host, port)
	check.Timeout = "5s"
	check.Interval = "5s"
	check.DeregisterCriticalServiceAfter = "20s"

	registration.Check = check

	if err := c.client.Agent().ServiceRegister(registration); err != nil {
		return err
	}
	c.localServiceID = id
	return nil
}

func (c ConsulServiceRegistry) Deregister() {
	_ = c.client.Agent().ServiceDeregister(c.localServiceID)
	c.localServiceID = ""
}

func NewConsulServiceRegistry(host string, port int, token string) (ConsulServiceRegistry, error) {
	var csr ConsulServiceRegistry
	if len(host) < 3 {
		return csr, errors.New("check host")
	}
	if port <= 0 || port > 65535 {
		return csr, errors.New("check port, port should between 1 and 65535")
	}
	config := api.DefaultConfig()
	config.Address = host + ":" + strconv.Itoa(port)
	config.Token = token
	client, err := api.NewClient(config)
	if err != nil {
		return csr, err
	}
	csr.client = *client
	return csr, nil
}
