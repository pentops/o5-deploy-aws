package deployer

import (
	"fmt"
)

var ErrDeploymentNotFound = fmt.Errorf("deployment not found")
var ErrStackNotFound = fmt.Errorf("stack not found")
