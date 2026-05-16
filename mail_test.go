package genelet

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSmtpRejectsHeaderInjection(t *testing.T) {
	smtp := &Smtp{
		Address: "127.0.0.1:1",
		From:    "sender@example.test",
		To:      []string{"recipient@example.test"},
	}

	err := smtp.Send(map[string]string{
		"To":      "recipient@example.test",
		"Subject": "hello\r\nBcc: attacker@example.test",
	}, "body")
	if gerr, ok := err.(Gerror); !ok || gerr.Code != 2066 {
		t.Fatalf("Smtp.Send header injection error = %#v, want 2066 Gerror", err)
	}
}

func TestSmtpsslVerifiesCertificatesByDefault(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	smtp := &Smtpssl{
		Address: strings.TrimPrefix(server.URL, "https://"),
		From:    "sender@example.test",
		To:      []string{"recipient@example.test"},
	}
	err := smtp.Send(map[string]string{
		"To":      "recipient@example.test",
		"Subject": "hello",
	}, "body")
	if err == nil {
		t.Fatal("Smtpssl.Send succeeded with an untrusted certificate")
	}
}
