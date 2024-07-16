package objects

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/simplecontainer/smr/pkg/httpcontract"
	"io"
	"net/http"
)

func SendRequest(client *http.Client, URL string, method string, data map[string]string) *httpcontract.ResponseOperator {
	var req *http.Request
	var marshaled []byte
	var err error

	if data != nil {
		marshaled, err = json.Marshal(data)

		if err != nil {
			return &httpcontract.ResponseOperator{
				HttpStatus:       0,
				Explanation:      "failed to marshal data for sending request",
				ErrorExplanation: err.Error(),
				Error:            true,
				Success:          false,
				Data:             nil,
			}
		}

		req, err = http.NewRequest(method, URL, bytes.NewBuffer(marshaled))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, URL, nil)
		req.Header.Set("Content-Type", "application/json")
	}

	if err != nil {
		return &httpcontract.ResponseOperator{
			HttpStatus:       0,
			Explanation:      "failed to craft request",
			ErrorExplanation: err.Error(),
			Error:            true,
			Success:          false,
			Data:             nil,
		}
	}

	resp, err := client.Do(req)

	if err != nil {
		return &httpcontract.ResponseOperator{
			HttpStatus:       0,
			Explanation:      "failed to connect to the smr-agent",
			ErrorExplanation: err.Error(),
			Error:            true,
			Success:          false,
			Data:             nil,
		}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &httpcontract.ResponseOperator{
			HttpStatus:       0,
			Explanation:      "invalid response from the smr-agent",
			ErrorExplanation: err.Error(),
			Error:            true,
			Success:          false,
			Data:             nil,
		}
	}

	var response httpcontract.ResponseOperator
	err = json.Unmarshal(body, &response)

	if err != nil {
		return &httpcontract.ResponseOperator{
			HttpStatus:       0,
			Explanation:      "failed to unmarshal body response from smr-agent",
			ErrorExplanation: generateResponse(URL, method, marshaled, body),
			Error:            true,
			Success:          false,
			Data:             nil,
		}
	}

	return &response
}

func generateResponse(URL string, method string, data []byte, body []byte) string {
	return fmt.Sprintf("URL: %s\nMETHOD: %s\nSEND_DATA: \n\n%s\nRESPONSE:\n\n%s", URL, method, string(data), string(body))
}
