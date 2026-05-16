package genelet

import (
	"net"
	"net/smtp"
)

type Smtp struct {
	Username string
	Password string
	Address  string
	From     string
	Headers  map[string]string
	To       []string
}

func (self *Smtp) Send(headers map[string]string, content string) error {
	for key, val := range self.Headers {
		if headers[key] == "" {
			headers[key] = val
		}
	}
	if headers["From"] == "" {
		headers["From"] = self.From
	}
	if headers["Subject"] == "" {
		return Err(2061)
	}
	if self.To == nil {
		if headers["To"] == "" {
			return Err(2062)
		}
		to, err := parseMailRecipients([]string{headers["To"]})
		if err != nil {
			return err
		}
		self.To = to
	}
	if headers["To"] == "" {
		headers["To"] = self.To[0]
	}
	if err := validateMailHeaders(headers); err != nil {
		return err
	}
	message := ""
	for k, v := range headers {
		message += k + ": " + v + "\r\n"
	}
	message += "\r\n" + content

	host, _, _ := net.SplitHostPort(self.Address)
	auth := smtp.PlainAuth("", self.Username, self.Password, host)
	err := smtp.SendMail(self.Address, auth, self.From, self.To, []byte(message))
	if err != nil {
		return err
	}
	return nil
}
