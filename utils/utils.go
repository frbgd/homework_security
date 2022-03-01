package utils

import "net/http"

func CopyHeaders(to, from http.Header) {
	for h, vv := range from {
		for _, v := range vv {
			to.Add(h, v)
		}
	}
}
