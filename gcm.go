package genelet

import ()

type Gcm struct {
	Api_key          string
	Registration_ids string
	Delay_while_idle string
	Time_to_live     string
	Collapse_key     string
}

func (self *Gcm) Send(body string) error {
	return nil
}
