package protoread

import (
	"fmt"
	"testing"
)

func TestFindTokenError(t *testing.T) {
	tc := []struct {
		data string
		err  string
	}{
		{
			err: "proto: syntax error (line 2:17): unexpected token",
			data: `{
			"blobstores": {
			"name": "hooligan-sandbox-s3"
			},
			"databases": [
			{`,
		},
	}

	for _, tt := range tc {
		err := findTokenError([]byte(tt.data), fmt.Errorf("find token: %s", tt.err))
		if err == nil {
			t.Error("expected error, got nil")
		}

		t.Log(err)
	}
}
