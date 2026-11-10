package auth

import (
	"testing"

	"github.com/alexedwards/argon2id"
)

func TestHash(t *testing.T) {
	case1Hash, _ := HashPassword("123456")
	case2Hash, _ := HashPassword("thequickbrownfoxjumpsoverthelazydog")
	case3Hash, _ := HashPassword("justs0m3l3tt3rs4ndnum63rs")
	cases := []struct {
		password string
		hash     string
		err      error
	}{
		{
			password: "123456",
			hash:     case1Hash,
		},
		{
			password: "thequickbrownfoxjumpsoverthelazydog",
			hash:     case2Hash,
		},
		{
			password: "justs0m3l3tt3rs4ndnum63rs",
			hash:     case3Hash,
		},
	}
	for _, c := range cases {
		hashMatch, err := argon2id.ComparePasswordAndHash(c.password, c.hash)
		if err != nil {
			t.Errorf("Got error %v\nWhile checking hash on password: %s", err, c.password)
			return
		}
		if !hashMatch {
			t.Errorf("Hash for %s did not match hash", c.password)
			return
		}
	}
}

func TestHashCheck(t *testing.T) {
	testHash, _ := HashPassword("thisShouldMatch1Case")
	cases := []struct {
		password    string
		correctHash string
		match       bool
	}{
		{
			password:    "thisShouldMatch1Case",
			correctHash: testHash,
			match:       true,
		},
		{
			password:    "thisWillNotMatch",
			correctHash: testHash,
			match:       false,
		},
	}
	for _, c := range cases {
		doesMatch, err := CheckPasswordHash(c.password, c.correctHash)
		if err != nil {
			t.Errorf("Error checking password hash for password: %s", c.password)
			return
		}
		if doesMatch != c.match {
			t.Errorf("Actual did not match for password %s.\nExpected: %v, Got: %v", c.password, c.match, doesMatch)
			return
		}
	}
}
