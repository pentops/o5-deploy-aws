package cf

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/awslabs/goformation/v7/cloudformation"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func String(str string) *string {
	return &str
}

func Stringf(str string, params ...interface{}) *string {
	return String(fmt.Sprintf(str, params...))
}

func Int(i int) *int {
	return &i
}

func Bool(b bool) *bool {
	return &b

}

var reUnsafe = regexp.MustCompile(`[^a-zA-Z0-9]`)

func ResourceName(name string, rr cloudformation.Resource) string {
	resourceType := strings.ReplaceAll(rr.AWSCloudFormationType(), "::", "")
	resourceType = strings.TrimPrefix(resourceType, "AWS_")
	name = reResourceUnsafe.ReplaceAllString(name, "")
	return fmt.Sprintf("%s%s", resourceType, name)
}

func CleanParameterName(unsafes ...string) string {
	titleCase := cases.Title(language.English)
	outParts := []string{}
	for _, unsafe := range unsafes {
		safeString := reUnsafe.ReplaceAllString(unsafe, "_")
		parts := strings.Split(safeString, "_")
		for _, part := range parts {
			outParts = append(outParts, titleCase.String(part))
		}
	}
	safeString := strings.Join(outParts, "")
	return safeString
}
