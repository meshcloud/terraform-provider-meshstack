package client

import (
	"net/http"
)

func isSuccessHTTPStatus(resp *http.Response) bool {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false
	}

	return true
}
