package genelet

import (
	"crypto/tls"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
)

type Smtpssl struct {
	Username string
	Password string
	Address  string
	Headers  map[string]string
	From     string
	To       []string
}

func (self *Smtpssl) Send(headers map[string]string, content string) error {
	for key, val := range self.Headers {
		if headers[key] == "" {
			headers[key] = val
		}
	}
	if headers["From"] == "" {
		headers["From"] = self.From
	}
	if headers["Subject"] == "" {
		return Err(2065)
	}
	if self.To == nil {
		if headers["To"] == "" {
			return Err(2062)
		}
		self.To = strings.Split(headers["To"], ",")
	}
	if headers["To"] == "" {
		headers["To"] = self.To[0]
	}

	message := ""
	for k, v := range headers {
		message += k + ": " + v + "\r\n"
	}
	message += "\r\n" + content

	host, _, _ := net.SplitHostPort(self.Address)
	auth := smtp.PlainAuth("", self.Username, self.Password, host)

	tlsconfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         host,
	}
	// Here is the key, you need to call tls.Dial instead of smtp.Dial
	// for smtp servers running on 465 that require an ssl connection
	// from the very beginning (no starttls)
	conn, err := tls.Dial("tcp", self.Address, tlsconfig)
	if err != nil {
		return err
	}

	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	if err = c.Auth(auth); err != nil {
		return err
	}
	if err = c.Mail(mail.Address{Address: self.From}.Address); err != nil {
		return err
	}
	if err = c.Rcpt(mail.Address{Address: self.To[0]}.Address); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	_, err = w.Write([]byte(message))
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}
	c.Quit()
	return nil
}
