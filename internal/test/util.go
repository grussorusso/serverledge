package test

import (
	"encoding/base64"
	"fmt"
	"github.com/grussorusso/serverledge/internal/cli"
	"github.com/grussorusso/serverledge/internal/function"
)

func initializeExamplePyFunction() (*function.Function, error) {
	srcPath := "../../examples/inc.py"
	srcContent, err := cli.ReadSourcesAsTar(srcPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read python sources %s as tar: %v", srcPath, err)
	}
	encoded := base64.StdEncoding.EncodeToString(srcContent)
	f := function.Function{
		Name:            "inc",
		Runtime:         "python310",
		MemoryMB:        20,
		CPUDemand:       1.0,
		Handler:         "inc.handler", // on python, for now is needed file name and handler name!!
		TarFunctionCode: encoded,
		Signature: function.NewSignature().
			AddInput("input", function.Int{}).
			AddOutput("result", function.Int{}).
			Build(),
	}

	return &f, nil
}

func initializeExampleJSFunction() (*function.Function, error) {
	srcPath := "../../examples/inc.js"
	srcContent, err := cli.ReadSourcesAsTar(srcPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read js sources %s as tar: %v", srcPath, err)
	}
	encoded := base64.StdEncoding.EncodeToString(srcContent)
	f := function.Function{
		Name:            "inc",
		Runtime:         "nodejs17ng",
		MemoryMB:        50,
		CPUDemand:       1.0,
		Handler:         "inc", // for js, only the file name is needed!!
		TarFunctionCode: encoded,
		Signature: function.NewSignature().
			AddInput("input", function.Int{}).
			AddOutput("result", function.Int{}).
			Build(),
	}

	return &f, nil
}

func initializePyFunction(name string, handler string, sign *function.Signature) (*function.Function, error) {
	srcPath := "../../examples/" + name + ".py"
	srcContent, err := cli.ReadSourcesAsTar(srcPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read js sources %s as tar: %v", srcPath, err)
	}
	encoded := base64.StdEncoding.EncodeToString(srcContent)
	f := function.Function{
		Name:            name,
		Runtime:         "python310",
		MemoryMB:        600,
		CPUDemand:       1.0,
		Handler:         fmt.Sprintf("%s.%s", name, handler), // on python, for now is needed file name and handler name!!
		TarFunctionCode: encoded,
		Signature:       sign,
	}
	return &f, nil
}

// initializeSameFunctionSlice is used to easily initialize a function array with one single function
func initializeSameFunctionSlice(length int, jsOrPy string) (*function.Function, []*function.Function, error) {
	var f *function.Function
	var err error
	if jsOrPy == "js" {
		f, err = initializeExampleJSFunction()
	} else if jsOrPy == "py" {
		f, err = initializeExamplePyFunction()
	} else {
		return nil, nil, fmt.Errorf("you can only choose from js or py (or custom runtime...)")
	}
	if err != nil {
		return f, nil, err
	}
	fArr := make([]*function.Function, length)
	for i := 0; i < length; i++ {
		fArr[i] = f
	}
	return f, fArr, nil
}
