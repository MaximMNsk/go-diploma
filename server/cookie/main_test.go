package cookie

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHash(t *testing.T) {
	type args struct {
		inputToken string
	}

	type want struct {
		userID int
	}

	tests := []struct {
		name      string
		arguments args
		want      want
	}{
		{
			name:      `Test explode user ID from token`,
			arguments: args{inputToken: `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MDc3MjYwMDMsIlVzZXJJRCI6MX0.ThLBJ6ZgY9sn8wBKBOAu1SfTX5kTYG5x620ziwUotYM`},
			want:      want{userID: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := GetUserID(tt.arguments.inputToken)
			assert.Equal(t, data, tt.want.userID)
		})
	}
}
