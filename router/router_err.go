package router

import "fmt"

type AddHandlerRepeatErr struct {
	HandlerName string
}

func (a AddHandlerRepeatErr) Error() string {
	return fmt.Sprintf("add handler repeatedly error:%s", a.HandlerName)
}

type AddDispatchRepeatErr struct {
	Topic string
}

func (a AddDispatchRepeatErr) Error() string {
	return fmt.Sprintf("add dispatch repeatedly error:%s", a.Topic)
}
