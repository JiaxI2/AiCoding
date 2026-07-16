package mcpcontrol

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type rpcError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error,omitempty"`
}

type initializeResult struct {
	ProtocolVersion string                     `json:"protocolVersion"`
	Capabilities    map[string]json.RawMessage `json:"capabilities"`
}

type rpcCaller func(int, string, interface{}) (json.RawMessage, error)

func Probe(ctx context.Context, endpoint Endpoint) ProbeResult {
	result := ProbeResult{ID: endpoint.ID, Transport: endpoint.Transport, OK: false}
	var err error
	switch endpoint.Transport {
	case "stdio":
		err = probeStdio(ctx, endpoint, &result)
	case "streamable-http":
		err = probeHTTP(ctx, endpoint, &result)
	default:
		err = fmt.Errorf("unsupported MCP transport: %s", endpoint.Transport)
	}
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
		return result
	}
	result.OK = true
	return result
}

func ProbeAll(ctx context.Context, endpoints []Endpoint, maxParallel int) []ProbeResult {
	if maxParallel <= 0 {
		maxParallel = 4
	}
	results := make([]ProbeResult, len(endpoints))
	semaphore := make(chan struct{}, maxParallel)
	var wait sync.WaitGroup
	for index, endpoint := range endpoints {
		index := index
		endpoint := endpoint
		wait.Add(1)
		go func() {
			defer wait.Done()
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				results[index] = ProbeResult{
					ID:        endpoint.ID,
					Transport: endpoint.Transport,
					OK:        false,
					Errors:    []string{ctx.Err().Error()},
				}
				return
			}
			results[index] = Probe(ctx, endpoint)
		}()
	}
	wait.Wait()
	return results
}

func probeStdio(parent context.Context, endpoint Endpoint, result *ProbeResult) error {
	if endpoint.Command == "" {
		return errors.New("stdio MCP command is empty")
	}
	timeout := endpoint.StartupTimeoutSec
	if timeout <= 0 {
		timeout = 30
	}
	ctx, cancel := context.WithTimeout(parent, time.Duration(timeout)*time.Second)
	defer cancel()

	command := exec.CommandContext(ctx, endpoint.Command, endpoint.Args...)
	command.Dir = endpoint.Cwd
	command.Env = mergeEnvironment(endpoint.Env)
	stdin, err := command.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		return err
	}
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	defer closeProcess(stdin, command, done)

	reader := bufio.NewReader(stdout)
	caller := func(id int, method string, params interface{}) (json.RawMessage, error) {
		if err := writeRPC(stdin, id, method, params); err != nil {
			return nil, err
		}
		response, err := readLineResponse(reader, id)
		if err != nil {
			if text := tail(stderr.String(), 2000); text != "" {
				return nil, fmt.Errorf("%w: %s", err, text)
			}
			return nil, err
		}
		return response.Result, nil
	}

	raw, err := caller(1, "initialize", initializeParams())
	if err != nil {
		return err
	}
	initialized, err := parseInitialize(raw)
	if err != nil {
		return err
	}
	applyInitialize(result, initialized)
	if err := writeNotification(stdin, "notifications/initialized", map[string]interface{}{}); err != nil {
		return err
	}
	return populateDiscovery(result, initialized.Capabilities, caller)
}

func probeHTTP(parent context.Context, endpoint Endpoint, result *ProbeResult) error {
	if endpoint.URL == "" {
		return errors.New("streamable HTTP MCP URL is empty")
	}
	timeout := endpoint.StartupTimeoutSec
	if timeout <= 0 {
		timeout = 30
	}
	ctx, cancel := context.WithTimeout(parent, time.Duration(timeout)*time.Second)
	defer cancel()
	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}
	headers := map[string]string{}
	if endpoint.BearerTokenEnvVar != "" {
		token := os.Getenv(endpoint.BearerTokenEnvVar)
		if token == "" {
			return fmt.Errorf("bearer token environment variable is not set: %s", endpoint.BearerTokenEnvVar)
		}
		headers["Authorization"] = "Bearer " + token
	}
	sessionID := ""
	protocol := ""
	defer func() {
		closeHTTPSession(client, endpoint.URL, headers, sessionID, protocol)
	}()

	call := func(id int, method string, params interface{}) (json.RawMessage, error) {
		response, responseHeaders, err := postRPC(ctx, client, endpoint.URL, headers, sessionID, protocol, id, method, params)
		if err != nil {
			return nil, err
		}
		if value := responseHeaders.Get("Mcp-Session-Id"); value != "" {
			sessionID = value
		}
		return response.Result, nil
	}

	raw, err := call(1, "initialize", initializeParams())
	if err != nil {
		return err
	}
	initialized, err := parseInitialize(raw)
	if err != nil {
		return err
	}
	protocol = initialized.ProtocolVersion
	applyInitialize(result, initialized)
	if err := postNotification(ctx, client, endpoint.URL, headers, sessionID, protocol, "notifications/initialized", map[string]interface{}{}); err != nil {
		return err
	}
	return populateDiscovery(result, initialized.Capabilities, call)
}

func initializeParams() map[string]interface{} {
	return map[string]interface{}{
		"protocolVersion": ProtocolVersion,
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]string{
			"name":    "aicoding-mcp-control",
			"version": "0.1.0",
		},
	}
}

func parseInitialize(raw json.RawMessage) (initializeResult, error) {
	var initialized initializeResult
	if err := json.Unmarshal(raw, &initialized); err != nil {
		return initializeResult{}, err
	}
	if initialized.ProtocolVersion == "" {
		return initializeResult{}, errors.New("initialize response has no protocolVersion")
	}
	return initialized, nil
}

func applyInitialize(result *ProbeResult, initialized initializeResult) {
	result.ProtocolVersion = initialized.ProtocolVersion
	result.Capabilities = CapabilitySummary{
		Tools:     hasCapability(initialized.Capabilities, "tools"),
		Resources: hasCapability(initialized.Capabilities, "resources"),
		Prompts:   hasCapability(initialized.Capabilities, "prompts"),
		Logging:   hasCapability(initialized.Capabilities, "logging"),
	}
	if initialized.ProtocolVersion != ProtocolVersion {
		result.Warnings = append(
			result.Warnings,
			"negotiated protocol version "+initialized.ProtocolVersion+" instead of "+ProtocolVersion,
		)
	}
}

func populateDiscovery(result *ProbeResult, capabilities map[string]json.RawMessage, call rpcCaller) error {
	nextID := 2
	if hasCapability(capabilities, "tools") {
		raw, err := call(nextID, "tools/list", map[string]interface{}{})
		if err != nil {
			return err
		}
		nextID++
		var listed struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		}
		if err := json.Unmarshal(raw, &listed); err != nil {
			return err
		}
		for _, tool := range listed.Tools {
			result.Tools = append(result.Tools, tool.Name)
		}
		sort.Strings(result.Tools)
		result.ToolCount = len(result.Tools)
	}
	if hasCapability(capabilities, "resources") {
		raw, err := call(nextID, "resources/list", map[string]interface{}{})
		if err != nil {
			return err
		}
		nextID++
		var listed struct {
			Resources []json.RawMessage `json:"resources"`
		}
		if err := json.Unmarshal(raw, &listed); err != nil {
			return err
		}
		result.ResourceCount = len(listed.Resources)
	}
	if hasCapability(capabilities, "prompts") {
		raw, err := call(nextID, "prompts/list", map[string]interface{}{})
		if err != nil {
			return err
		}
		var listed struct {
			Prompts []json.RawMessage `json:"prompts"`
		}
		if err := json.Unmarshal(raw, &listed); err != nil {
			return err
		}
		result.PromptCount = len(listed.Prompts)
	}
	return nil
}

func hasCapability(capabilities map[string]json.RawMessage, name string) bool {
	raw, ok := capabilities[name]
	return ok && string(raw) != "null"
}

func writeRPC(writer io.Writer, id int, method string, params interface{}) error {
	message := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	}
	return writeJSONLine(writer, message)
}

func writeNotification(writer io.Writer, method string, params interface{}) error {
	message := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	}
	return writeJSONLine(writer, message)
}

func writeJSONLine(writer io.Writer, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = writer.Write(data)
	return err
}

func readLineResponse(reader *bufio.Reader, id int) (rpcResponse, error) {
	expected := strconv.Itoa(id)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			return rpcResponse{}, err
		}
		var response rpcResponse
		if err := json.Unmarshal(bytes.TrimSpace(line), &response); err != nil {
			return rpcResponse{}, err
		}
		if string(response.ID) != expected {
			continue
		}
		if response.Error != nil {
			return rpcResponse{}, fmt.Errorf("MCP error %d: %s", response.Error.Code, response.Error.Message)
		}
		return response, nil
	}
}

func postRPC(
	ctx context.Context,
	client *http.Client,
	url string,
	headers map[string]string,
	sessionID string,
	protocol string,
	id int,
	method string,
	params interface{},
) (rpcResponse, http.Header, error) {
	payload, err := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	})
	if err != nil {
		return rpcResponse{}, nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return rpcResponse{}, nil, err
	}
	applyHTTPHeaders(request, headers, sessionID, protocol)
	response, err := client.Do(request)
	if err != nil {
		return rpcResponse{}, nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 8192))
		return rpcResponse{}, response.Header, fmt.Errorf("HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}
	parsed, err := readHTTPResponse(response.Body, response.Header.Get("Content-Type"), id)
	if err != nil {
		return rpcResponse{}, response.Header, err
	}
	return parsed, response.Header, nil
}

func postNotification(
	ctx context.Context,
	client *http.Client,
	url string,
	headers map[string]string,
	sessionID string,
	protocol string,
	method string,
	params interface{},
) error {
	payload, err := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	})
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	applyHTTPHeaders(request, headers, sessionID, protocol)
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 8192))
		return fmt.Errorf("HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func applyHTTPHeaders(request *http.Request, headers map[string]string, sessionID string, protocol string) {
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	if sessionID != "" {
		request.Header.Set("Mcp-Session-Id", sessionID)
	}
	if protocol != "" {
		request.Header.Set("MCP-Protocol-Version", protocol)
	}
}

func readHTTPResponse(reader io.Reader, contentType string, id int) (rpcResponse, error) {
	if strings.Contains(strings.ToLower(contentType), "text/event-stream") {
		return readSSEResponse(reader, id)
	}
	var response rpcResponse
	if err := json.NewDecoder(reader).Decode(&response); err != nil {
		return rpcResponse{}, err
	}
	if string(response.ID) != strconv.Itoa(id) {
		return rpcResponse{}, fmt.Errorf("MCP response id mismatch: got %s, want %d", string(response.ID), id)
	}
	if response.Error != nil {
		return rpcResponse{}, fmt.Errorf("MCP error %d: %s", response.Error.Code, response.Error.Message)
	}
	return response, nil
}

func readSSEResponse(reader io.Reader, id int) (rpcResponse, error) {
	expected := strconv.Itoa(id)
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 8*1024*1024)
	dataLines := []string{}
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if len(dataLines) == 0 {
				continue
			}
			var response rpcResponse
			if err := json.Unmarshal([]byte(strings.Join(dataLines, "\n")), &response); err != nil {
				return rpcResponse{}, err
			}
			dataLines = dataLines[:0]
			if string(response.ID) != expected {
				continue
			}
			if response.Error != nil {
				return rpcResponse{}, fmt.Errorf("MCP error %d: %s", response.Error.Code, response.Error.Message)
			}
			return response, nil
		}
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	if err := scanner.Err(); err != nil {
		return rpcResponse{}, err
	}
	return rpcResponse{}, errors.New("SSE stream ended before matching MCP response")
}

func closeHTTPSession(client *http.Client, url string, headers map[string]string, sessionID string, protocol string) {
	if sessionID == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return
	}
	applyHTTPHeaders(request, headers, sessionID, protocol)
	response, err := client.Do(request)
	if err == nil {
		response.Body.Close()
	}
}

func closeProcess(stdin io.Closer, command *exec.Cmd, done <-chan error) {
	_ = stdin.Close()
	select {
	case <-done:
		return
	case <-time.After(2 * time.Second):
		if command.Process != nil {
			_ = command.Process.Kill()
		}
		<-done
	}
}

func mergeEnvironment(overrides map[string]string) []string {
	values := map[string]string{}
	names := map[string]string{}
	for _, item := range os.Environ() {
		key, value, found := strings.Cut(item, "=")
		if !found {
			continue
		}
		normalized := key
		if runtime.GOOS == "windows" {
			normalized = strings.ToUpper(key)
		}
		names[normalized] = key
		values[normalized] = value
	}
	for key, value := range overrides {
		normalized := key
		if runtime.GOOS == "windows" {
			normalized = strings.ToUpper(key)
		}
		names[normalized] = key
		values[normalized] = value
	}
	output := make([]string, 0, len(values))
	for normalized, value := range values {
		output = append(output, names[normalized]+"="+value)
	}
	sort.Strings(output)
	return output
}

func tail(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len(value) <= limit {
		return value
	}
	return value[len(value)-limit:]
}
