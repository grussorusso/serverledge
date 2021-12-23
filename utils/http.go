package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
)

func PostJson(url string, body []byte) (*http.Response, error) {
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return resp, fmt.Errorf("Server response: %v", resp.Status)
	}
	return resp, nil
}

func PrintJsonResponse(resp io.ReadCloser) {
	defer resp.Close()
	body, _ := ioutil.ReadAll(resp)

	// print indented JSON
	var out bytes.Buffer
	json.Indent(&out, body, "", "\t")
	out.WriteTo(os.Stdout)
}
