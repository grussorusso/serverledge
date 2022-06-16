package logging

import "errors"

var NotExistingLog = errors.New("functionLog not exists")
var GeneralError = errors.New("could not send ExecReport")
