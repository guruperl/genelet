package genelet

import (
	"net/mail"
	"strings"
)

func validateMailHeaders(headers map[string]string) error {
	for k, v := range headers {
		if k == "" {
			return Err(2066, "empty mail header name")
		}
		for _, r := range k {
			if r <= 32 || r >= 127 || strings.ContainsRune("()<>@,;:\\\"/[]?={}", r) {
				return Err(2066, "invalid mail header name")
			}
		}
		if strings.ContainsAny(v, "\r\n") {
			return Err(2066, "invalid mail header value")
		}
	}
	return nil
}

func parseMailRecipients(values []string) ([]string, error) {
	out := make([]string, 0)
	for _, raw := range values {
		if raw == "" {
			continue
		}
		addrs, err := mail.ParseAddressList(raw)
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			out = append(out, addr.Address)
		}
	}
	if len(out) == 0 {
		return nil, Err(2062)
	}
	return out, nil
}
