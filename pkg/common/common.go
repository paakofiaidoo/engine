package common

import (
	"time"
)

type ApiResponse[T any] struct {
	Success   bool     `json:"success"`
	Timestamp int64    `json:"timestamp"`
	Message   string   `json:"message"`
	Errors    []string `json:"errors"`
	Data      T        `json:"data"`
}

func NewApiResponse() *ApiResponse[any] {
	return &ApiResponse[any]{
		Success:   true,
		Timestamp: time.Now().UnixMilli(),
		Errors:    []string{},
		Data:      nil,
	}
}

func NewResponse() *Resp {
	return &Resp{}
}

type Resp struct {
	Body ApiResponse[any] `json:"body"`
}
