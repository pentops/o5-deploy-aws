package aws_cf

import (
	"encoding/base64"
	"testing"
)

func TestParseRawMessage(t *testing.T) {

	realMessage := `U3RhY2tJZD0nYXJuOmF3czpjbG91ZGZvcm1hdGlvbjp1cy1lYXN0LTE6MDAxODQ0NDY3NDgwOnN0YWNrL2RldndlYi11c2VyYXV0aC82NDgyMThjMC02NDQ2LTExZWUtOWFmMS0wYTc2MWQyYzcwODMnClRpbWVzdGFtcD0nMjAyMy0xMS0xNVQwMDowNDozNS45NDZaJwpFdmVudElkPSc4ZmRmYTAwMC04MzRhLTExZWUtYWQwZi0wYTZhMzk1YjgzYTUnCkxvZ2ljYWxSZXNvdXJjZUlkPSdkZXZ3ZWItdXNlcmF1dGgnCk5hbWVzcGFjZT0nMDAxODQ0NDY3NDgwJwpQaHlzaWNhbFJlc291cmNlSWQ9J2Fybjphd3M6Y2xvdWRmb3JtYXRpb246dXMtZWFzdC0xOjAwMTg0NDQ2NzQ4MDpzdGFjay9kZXZ3ZWItdXNlcmF1dGgvNjQ4MjE4YzAtNjQ0Ni0xMWVlLTlhZjEtMGE3NjFkMmM3MDgzJwpQcmluY2lwYWxJZD0nQVJPQVFBM1BRSzRNSENGQkQzNlFCOm81LWRlcGxveS1hd3MtMTcwMDAwNjY3MycKUmVzb3VyY2VQcm9wZXJ0aWVzPSdudWxsJwpSZXNvdXJjZVN0YXR1cz0nVVBEQVRFX0lOX1BST0dSRVNTJwpSZXNvdXJjZVN0YXR1c1JlYXNvbj0nVXNlciBJbml0aWF0ZWQnClJlc291cmNlVHlwZT0nQVdTOjpDbG91ZEZvcm1hdGlvbjo6U3RhY2snClN0YWNrTmFtZT0nZGV2d2ViLXVzZXJhdXRoJwpDbGllbnRSZXF1ZXN0VG9rZW49JzIwZjE1Y2FlLThmOGQtNGY5ZC1hNWQ0LWJiZmFmMGUzNjM3Mi1pbmZyYS1taWdyYXRlJwo=`

	asBytes, err := base64.StdEncoding.DecodeString(realMessage)
	if err != nil {
		t.Fatal(err)
	}

	asFields, err := parseAWSRawMessage(asBytes)
	if err != nil {
		t.Fatal(err)
	}

	if asFields["StackName"] != "devweb-userauth" {
		t.Errorf("Expected devweb-userauth, got %s", asFields["StackName"])
	}

}
