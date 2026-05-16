package genelet

type Apns struct {
	Badge       int8
	Sound       string
	DeviceToken string
	Cert        string
	Key         string
	Passphrase  string
}

func (self *Apns) Send(body string) error {
	return nil
}
