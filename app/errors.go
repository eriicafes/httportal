package app

import "net/http"

type ClientError struct {
	error
	Message string
	Desc    string
	Status  int
}

func NewClientError(err error, message string) ClientError {
	return ClientError{error: err, Message: message, Status: http.StatusBadRequest}
}

func (ce ClientError) WithStatus(status int) ClientError {
	ce.Status = status
	return ce
}

func (ce ClientError) WithDesc(desc string) ClientError {
	ce.Desc = desc
	return ce
}

func (ce ClientError) Unwrap() error {
	return ce.error
}
