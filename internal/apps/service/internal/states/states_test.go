package states

import "testing"

func TestStateMachineLogic(t *testing.T) {

	sm, err := NewStateMachines()
	if err != nil {
		t.Fatalf("NewStateMachines: %v", err)
	}

	deploymentStr, err := sm.Deployment.PrintMermaid()
	if err != nil {
		t.Fatalf("PrintMermaid: %v", err)
	}
	t.Logf("Deployment: \n%s", deploymentStr)

	environmentStr, err := sm.Environment.PrintMermaid()
	if err != nil {
		t.Fatalf("PrintMermaid: %v", err)
	}

	t.Logf("Environment: \n%s", environmentStr)

	clusterStr, err := sm.Cluster.PrintMermaid()
	if err != nil {
		t.Fatalf("PrintMermaid: %v", err)
	}

	t.Logf("Cluster: \n%s", clusterStr)

	stackStr, err := sm.Stack.PrintMermaid()
	if err != nil {
		t.Fatalf("PrintMermaid: %v", err)
	}

	t.Logf("Stack: \n%s", stackStr)

}
