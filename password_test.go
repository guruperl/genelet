package genelet

import "testing"

func TestPasswordHashUsesBcrypt(t *testing.T) {
	hash, err := HashPassword("secret")
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckPasswordHash("secret", hash); err != nil {
		t.Fatalf("CheckPasswordHash returned %v", err)
	}
	if err := CheckPasswordHash("wrong", hash); err == nil {
		t.Fatal("CheckPasswordHash accepted wrong password")
	}
}
