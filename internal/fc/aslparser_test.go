package fc_test

import (
	"fmt"
	"github.com/grussorusso/serverledge/internal/fc"
	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/internal/test"
	"os"
	"testing"
)

func TestParsingSimple(t *testing.T) {
	f, err := test.InitializePyFunction("inc", "handler", function.NewSignature().
		AddInput("input", function.Int{}).
		AddOutput("result", function.Int{}).
		Build())
	if err != nil {
		t.FailNow()
	}
	f.SaveToEtcd()

	body, err := os.ReadFile("../../test/simple.json")
	if err != nil {
		t.Fatalf("unable to read file: %v", err)
	}
	src := string(body)
	comp, err := fc.FromASL("prova", src)
	if err != nil {
		fmt.Printf("%v\n", err)
		t.Fail()
	}
	fmt.Println(comp)

	comp.SaveToEtcd()

	fmt.Println(fc.GetAllFC())
}

func TestParsing(t *testing.T) {
	body, err := os.ReadFile("../../test/machine.json")
	if err != nil {
		t.Fatalf("unable to read file: %v", err)
	}
	src := string(body)
	comp, err := fc.FromASL("prova", src)
	if err != nil {
		fmt.Printf("%v", err)
		t.Fail()
	}
	fmt.Println(comp)
}
