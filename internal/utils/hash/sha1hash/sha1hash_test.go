package sha1hash

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestHash(t *testing.T) {
	type args struct {
		inputString string
	}

	type want struct {
		outputString    string
		outputStringLen int
	}

	tests := []struct {
		name      string
		arguments args
		want      want
	}{
		{
			name:      `Test hash`,
			arguments: args{inputString: `password`},
			want:      want{outputStringLen: 40, outputString: `5baa61e4c9b93f3f0682250b6cf8331b7ee68fd8`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := Hash(tt.arguments.inputString)
			require.NoError(t, err)
			assert.Equal(t, data, tt.want.outputString)
			assert.Equal(t, len(data), tt.want.outputStringLen)
		})
	}
}
