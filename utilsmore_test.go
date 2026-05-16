package genelet

import (
	"testing"
)

func TestGeneletUtils(t *testing.T) {
	str := "abcdefg_+hijk=="
	new_str := Stripchars("df+=", str)
	if "abceg_hijk" != new_str {
		t.Errorf("%s %s wanted", str, new_str)
	}
	//join1 := Joinstrings("|", str, new_str)
	//if "abcdefg_+hijk==|abceg_hijk" != join1 {
	//		t.Errorf("%s wanted", join1)
	//}
	x := []string{str, new_str, "abc"}
	if Grep(x, "abcZ") {
		t.Errorf("%s wrong matched", "abcZ")
	}
	if Grep(x, "abc") == false {
		t.Errorf("%s matched", "abc")
	}

	ip := "67.214.232.46"
	id := Ip2int(ip)
	if id != 1138157614 {
		t.Errorf("%s %d matched", ip, id)
	}
	ip = "67.214.232.45"
	id = Ip2int(ip)
	if id != 1138157613 {
		t.Errorf("%s %d matched", ip, id)
	}
	ip = "67.214.232.47"
	id = Ip2int(ip)
	if id != 1138157615 {
		t.Errorf("%s %d matched", ip, id)
	}

	ip = "209.173.53.167"
	id = Ip2int(ip)
	if id != 3517789607 {
		t.Errorf("%s %d matched", ip, id)
	}
	ip = "0:0:0:0:0:ffff:d1ad:35a7"
	id = Ip2int(ip)
	if id != 3517789607 {
		t.Errorf("%s %d matched", ip, id)
	}
	if got := Ip2int("not an ip"); got != 0 {
		t.Errorf("invalid IP = %d, want 0", got)
	}
	if got := Interface2String(true); got != "true" {
		t.Errorf("bool string = %q, want true", got)
	}
	if got := Interface2String(map[string]interface{}{"ok": true}); got != `{"ok":true}` {
		t.Errorf("map string = %q", got)
	}
}
