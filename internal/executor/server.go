package executor

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"io/ioutil"
)

const resultFile = "/tmp/_executor_result.json"
const paramsFile = "/tmp/_executor.params"
const fallbackAddressesFile = "/tmp/_executor_fallback_addresses.txt"

func readExecutionResult(resultFile string) string {
	content, err := ioutil.ReadFile(resultFile)
	if err != nil {
		log.Printf("%v", err)
		return ""
	}

	return string(content)
}

func InvokeHandler(w http.ResponseWriter, r *http.Request) {
	// Parse request
	reqDecoder := json.NewDecoder(r.Body)
	req := &InvocationRequest{}
	err := reqDecoder.Decode(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Set environment variables
	os.Setenv("RESULT_FILE", resultFile)
	os.Setenv("HANDLER", req.Handler)
	os.Setenv("HANDLER_DIR", req.HandlerDir)
	os.Setenv("ID", req.Id)
	params := req.Params
	if params == nil {
		os.Setenv("PARAMS_FILE", "")
	} else {
		paramsB, _ := json.Marshal(req.Params)
		err := os.WriteFile(paramsFile, paramsB, 0644)
		if err != nil {
			log.Printf("Could not write parameters to %s", paramsFile)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		os.Setenv("PARAMS_FILE", paramsFile)
	}

	// Exec handler process
	cmd := req.Command
	if cmd == nil || len(cmd) < 1 {
		// this request is either invalid or uses a custom runtime
		// in the latter case, we find the command in the env
		customCmd, ok := os.LookupEnv("CUSTOM_CMD")
		if !ok {
			log.Printf("Invalid request!")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		cmd = strings.Split(customCmd, " ")
	}
	var resp *InvocationResult
	execCmd := exec.Command(cmd[0], cmd[1:]...)
	out, err := execCmd.CombinedOutput()
	if err != nil {
		log.Printf("cmd.Run() failed with %s\n", err)
		fmt.Printf("Function output:\n%s\n", string(out)) // TODO: do something with output
		resp = &InvocationResult{Success: false}
	} else {
		result := readExecutionResult(resultFile)

		resp = &InvocationResult{true, result, 0, ""}
		fmt.Printf("Function output:\n%s\n", string(out)) // TODO: do something with output
	}

	w.Header().Set("Content-Type", "application/json")
	respBody, _ := json.Marshal(resp)
	_, err = w.Write(respBody)
	if err != nil {
		fmt.Println("A migration has occurred.")
		// Build a migraion response struct
		var mresp = &MigrationResult{}
		mresp.Id = req.Id
		mresp.Result = resp.Result
		mresp.Success = true
		respBody, _ = json.Marshal(mresp)
		// Acquire the fallback IPs
		tmpFile, err := os.Open(fallbackAddressesFile)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		scanner := bufio.NewScanner(tmpFile)
		scanner.Split(bufio.ScanLines)
		var fallbackAddresses []string
		for scanner.Scan() {
			fallbackAddresses = append(fallbackAddresses, scanner.Text())
		}
		tmpFile.Close()
		// Send the response to the IPs
		for _, ip := range fallbackAddresses {
			url := fmt.Sprintf("http://%s:%d/receiveResultAfterMigration", ip, 1323)
			_, err = http.Post(url, "application/json", bytes.NewBuffer(respBody))
			if err != nil {
				fmt.Println("ERR: Could not send the response to ", ip, "\n-> ", err)
			} else {
				fmt.Println("\t...Response sent to ", ip)
				break
			}
		}
	}
}

func GetFallbackAddresses(w http.ResponseWriter, r *http.Request) {
	// Parse request
	reqDecoder := json.NewDecoder(r.Body)
	req := &FallbackAcquisitionRequest{}
	err := reqDecoder.Decode(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Save the fallback addresses locally
	fallbackAddresses := req.FallbackAddresses
	resp := &FallbackAcquisitionResult{true}
	tmpFile, err := os.Create(fallbackAddressesFile) // Create a temporary copy file to transfer it to the container
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, ip := range fallbackAddresses {
		fmt.Fprintln(tmpFile, ip)
	}
	tmpFile.Close()

	w.Header().Set("Content-Type", "application/json")
	respBody, _ := json.Marshal(resp)
	w.Write(respBody)
}
