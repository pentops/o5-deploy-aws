package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/pentops/o5-deploy-aws/cf"
	"github.com/pentops/o5-deploy-aws/protoread"
	"github.com/pentops/o5-go/application/v1/application_pb"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
)

func main() {
	envFilename := flag.String("env", "", "environment file")
	appFilename := flag.String("app", "", "application file")
	flag.Parse()

	if *envFilename == "" {
		fmt.Fprintln(os.Stderr, "missing environment file (-env)")
		os.Exit(1)
	}

	if *appFilename == "" {
		fmt.Fprintln(os.Stderr, "missing application file (-app)")
		os.Exit(1)
	}

	app := &application_pb.Application{}
	if err := protoread.ParseFile(*appFilename, app); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	env := &environment_pb.Environment{}
	if err := protoread.ParseFile(*envFilename, env); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	built, err := cf.BuildCloudformation(app, env)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	yaml, err := built.Template().YAML()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	fmt.Println(string(yaml))

}
