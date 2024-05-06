// /*** Based on https://github.com/enginyoyen/aslparser ***/
package fc

//package fc
//
//import (
//	"github.com/grussorusso/serverledge/internal/fc/static"
//	"github.com/xeipuuv/gojsonschema"
//)
//
//// Loads the state-machine JSON file from provided path
//// and validates it against states-language schema
//// strict argument defines whether Resource name must be AWS ARN pattern or not
//// See https://states-language.net/spec.html
//func Validate(payload []byte, strict bool) (*gojsonschema.Result, error) {
//	result, err := validateSchema(payload, strict)
//	if err != nil {
//		return result, err
//	}
//	return result, nil
//}
//
//func validateSchema(payload []byte, strict bool) (*gojsonschema.Result, error) {
//	stateMachineSchema, assetError := stateMachineSchema(strict)
//	if assetError != nil {
//		return nil, assetError
//	}
//	schemaLoader := gojsonschema.NewStringLoader(string(stateMachineSchema))
//	documentLoader := gojsonschema.NewStringLoader(string(payload))
//
//	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
//	return result, err
//}
//
//func stateMachineSchema(strict bool) ([]byte, error) {
//	if strict {
//		return static.Asset("schemas/state-machine-strict-arn.json")
//	} else {
//		return static.Asset("schemas/state-machine.json")
//	}
//
//}
