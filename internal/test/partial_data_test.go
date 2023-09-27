package test

import (
	"encoding/json"
	"fmt"
	"github.com/grussorusso/serverledge/internal/fc"
	u "github.com/grussorusso/serverledge/utils"
	"testing"
)

func TestPartialDataMarshaling(t *testing.T) {
	data := make(map[string]interface{})
	data["prova"] = "testo"
	data["num"] = 2
	data["list"] = []string{"uno", "due", "tre"}
	partialData := fc.PartialData{
		ReqId:    fc.ReqId("abc"),
		ForNode:  "fai13p102",
		FromNode: "120e8d12d",
		Data:     data,
	}
	marshal, errMarshal := json.Marshal(partialData)
	u.AssertNilMsg(t, errMarshal, "error during marshaling")
	var retrieved fc.PartialData
	errUnmarshal := json.Unmarshal(marshal, &retrieved)
	u.AssertNilMsg(t, errUnmarshal, "failed composition unmarshal")

	u.AssertTrueMsg(t, retrieved.Equals(partialData), fmt.Sprintf("retrieved partialData is not equal to initial partialData. Retrieved:\n%s,\nExpected:\n%s ", retrieved.String(), partialData.String()))
}
