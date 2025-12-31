package client

import "fmt"

type StatusError struct {
	URL  string
	Code int
}

func (e StatusError) Error() string {
	if e.URL == "" {
		return fmt.Sprintf("http status %d", e.Code)
	}
	return fmt.Sprintf("http status %d for %s", e.Code, e.URL)
}

func IsStatus(err error, codes ...int) bool {
	var statusErr StatusError
	switch v := err.(type) {
	case StatusError:
		statusErr = v
	case *StatusError:
		statusErr = *v
	default:
		return false
	}
	for _, code := range codes {
		if statusErr.Code == code {
			return true
		}
	}
	return false
}
