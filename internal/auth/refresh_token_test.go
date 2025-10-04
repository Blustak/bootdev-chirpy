package auth_test

import(
	"github.com/Blustak/bootdev-chirpy/internal/auth"
	"testing"
)

func TestMakeRefreshToken(t *testing.T) {
	tests := []struct {
		name    string // description of this test case
		wantErr bool
	}{
        {
        name: "test case",
        wantErr: false,
    },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := auth.MakeRefreshToken()
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("MakeRefreshToken() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("MakeRefreshToken() succeeded unexpectedly")
			}
			// TODO: update the condition below to compare got with tt.want.
			if got == "" {
				t.Errorf("MakeRefreshToken() = wanted non-empty string, got empty string.")
			}
		})
	}
}

