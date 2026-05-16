package genelet

import (
	"testing"
)

var CRYPTEXT = "12345678901234567890"
var text = "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz"

func TestScoder(t *testing.T) {
	encode := EncodeScoder(text, CRYPTEXT)
	decode := DecodeScoder(encode, CRYPTEXT)
	if encode != "ukscLf6PQBEi+Vo9mHPWqRT/Uj/IUrQWeNo+gPxcBsuQWSLvtGVE9XckwY5bEBXkrWSxIpkK61z9sDsHNao/RMF+lQCXAufYsUoDNOP8" {
		t.Errorf("%s got encode", encode)
	}
	if decode != text {
		t.Errorf("%s got decoded", decode)
	}
}
