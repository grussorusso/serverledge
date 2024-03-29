package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
	defer func(resp io.ReadCloser) {
		err := resp.Close()
		if err != nil {
			fmt.Printf("Error while closing JSON reader: %s\n", err)
		}
	}(resp)
	body, _ := io.ReadAll(resp)

	// print indented JSON
	var out bytes.Buffer
	err := json.Indent(&out, body, "", "\t")
	if err != nil {
		fmt.Printf("Error while indenting JSON: %s\n", err)
		return
	}
	_, err = out.WriteTo(os.Stdout)
	if err != nil {
		fmt.Printf("Error while writing indented JSON to stdout: %s\n", err)
		return
	}
}
