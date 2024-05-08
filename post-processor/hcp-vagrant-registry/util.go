package hcpvagrantregistry

import (
	"fmt"

	"github.com/go-openapi/runtime"
	"github.com/hashicorp/hcp-sdk-go/clients/cloud-shared/v1/models"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
)

type hcpErrorResponse interface {
	IsSuccess() bool
	IsRedirect() bool
	IsClientError() bool
	IsServerError() bool
	IsCode(int) bool
	Code() int
	GetPayload() *models.GoogleRPCStatus
}

func isErrorUnexpected(err error, state multistep.StateBag) bool {
	if err == nil {
		return false
	}

	if _, ok := errorResponse(err); !ok {
		state.Put("error", fmt.Errorf("Unexpected client error: %s", err))
		return true
	}

	return false
}

func errorResponse(err error) (hcpErrorResponse, bool) {
	if err == nil {
		return nil, false
	}

	if val, ok := err.(*runtime.APIError); ok {
		if resp, ok := val.Response.(hcpErrorResponse); ok {
			return resp, true
		}
	}

	if resp, ok := err.(hcpErrorResponse); ok {
		return resp, true
	}

	return nil, false
}

func errorStatus(err error) (*models.GoogleRPCStatus, bool) {
	if val, ok := errorResponse(err); ok {
		return val.GetPayload(), true
	}

	return nil, false
}

func errorResponseMsg(err error) (string, bool) {
	if val, ok := errorStatus(err); ok {
		msg := val.Message
		if msg == "" {
			msg = "Unexpected error encountered"
		}

		return msg, true
	}

	return "", false
}
