package main

import (
	"github.com/VladislavPav/trigger-hook/contracts"
	"github.com/VladislavPav/trigger-hook/domain"
)

func NewTransportAmqp() *amqpTransport {
	return &amqpTransport{}
}

type amqpTransport struct {
	contracts.SendingTransportInterface
}

func (amqpTransport) Send(task *domain.Task) bool {
	return true
}
