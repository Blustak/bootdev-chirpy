package auth_test

import (
	"testing"

	"github.com/Blustak/bootdev-chirpy/internal/auth"
	"github.com/alexedwards/argon2id"
)

func TestHashPassword(t *testing.T) {
	longString := func() string {
		s := ""
		for range 64 * 1024 {
			s = s + "aA1"
		}
		return s
	}()
	cases := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "basic test",
			password: "foobar",
			wantErr:  false,
		},
		{
			name:     "empty test",
			password: "",
			wantErr:  true,
		}, {
			name:     "long string test",
			password: longString,
			wantErr:  false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, gotErr := auth.HashPassword(c.password)
			if gotErr != nil {
				if !c.wantErr {
					t.Errorf("HashPassword() failed: %v", gotErr)
                }
					return
			}
            if c.wantErr {
                t.Fatal("HashPassword succeeded unexpectedly")
            }
			ok, err := argon2id.ComparePasswordAndHash(c.password, got)
            if err != nil {
                if !c.wantErr {
                    t.Errorf("HashPassword() failed:%v", err)
                }
                return
            }
            if c.wantErr {
                t.Fatal("ComparePasswordAndHash succeeded unexpectedly")
            }
			if !ok {
				t.Logf("failed[%s]:hash did not pass check.\n", c.name)
				t.Fail()
			}
		})
	}
}

func TestCheckPasswordHash(t *testing.T) {
	makeHash := func(s string) string {
		res, err := argon2id.CreateHash(s, &auth.Params)
		if err != nil {
			t.Fatalf("Couldn't hash string %s\nerr:%v\n", s, err)
		}
		return res
	}
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		password string
		hash     string
		want     bool
		wantErr  bool
	}{
		{
			name:     "basic test",
			password: "foobar",
			hash:     makeHash("foobar"),
			want:     true,
			wantErr:  false,
		},
		{
			name:     "basic negation test",
			password: "foobar",
			hash:     makeHash("fooBaz"),
			want:     false,
			wantErr:  false,
		},
		{
			name:     "empty password test",
			password: "",
			hash:     makeHash(""),
			want:     false,
			wantErr:  true,
		},
		{
			name:     "empty check test",
			password: "foobar",
			hash:     "",
			want:     false,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := auth.CheckPasswordHash(tt.password, tt.hash)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("CheckPasswordHash() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("CheckPasswordHash() succeeded unexpectedly")
			}
			if tt.want != got {
				t.Errorf("CheckPasswordHash() = %v, want %v", got, tt.want)
			}
		})
	}
}
