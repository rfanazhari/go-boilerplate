package service

import (
	"context"
	"errors"

	domainevent "github.com/rfanazhari/go-boilerplate/domain/event"
	"github.com/rfanazhari/go-boilerplate/pkg/errs"
)

var errBrokerDisabled = errors.New("rabbitmq is not configured")

type UnavailablePublisher struct{}

func (UnavailablePublisher) PublishTaskCompleted(context.Context, domainevent.TaskCompleted) error {
	return errs.New(errs.CodeEventUnavailable, errs.KindUnavailable, errBrokerDisabled)
}
