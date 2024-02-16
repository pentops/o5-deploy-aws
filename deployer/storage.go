package deployer

import (
	"fmt"
)

var DeploymentNotFoundError = fmt.Errorf("deployment not found")
var StackNotFoundError = fmt.Errorf("stack not found")
