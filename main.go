package main

import (
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	json "github.com/json-iterator/go"
)

var (
	host  = os.Getenv("logstash_host")
	port  = os.Getenv("logstash_port")
	token = os.Getenv("token")
)

func main() {
	lambda.Start(handle)
}

func handle(event events.CloudwatchLogsData) error {
	return processAll(event.LogGroup, event.LogStream, event.LogEvents)
}

func processAll(group, stream string, logs []events.CloudwatchLogsLogEvent) error {
	addr := host + ":" + port
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	for _, log := range logs {
		raw := logMessage(group, stream, log)
		if raw == nil {
			continue
		}

		if _, err := conn.Write(raw); err != nil {
			return err
		}
	}

	return nil
}

func logMessage(group, stream string, event events.CloudwatchLogsLogEvent) []byte {
	if strings.Contains(event.Message, "START RequestId") ||
		strings.Contains(event.Message, "END RequestId") ||
		strings.Contains(event.Message, "REPORT RequestId") {
		return nil
	}

	funcName := functionName(group)
	funcVersion, err := lambdaVersion(stream)
	if err != nil {
		return nil
	}

	msg := log{
		Stream:        stream,
		Group:         group,
		LambdaName:    funcName,
		Type:          "cloudwatch",
		Token:         token,
		Message:       []byte(event.Message),
		LambdaVersion: funcVersion,
	}

	raw, err := json.Marshal(msg)
	if err != nil {
		return nil
	}

	return raw
}

func lambdaVersion(stream string) (int, error) {
	start := strings.Index(stream, "[")
	end := strings.Index(stream, "]")

	return strconv.Atoi(stream[start:end])
}

func functionName(group string) string {
	return strings.Split(group, "/")[len(group)-1]
}

type log struct {
	Stream        string `json:"stream"`
	Group         string `json:"group"`
	LambdaName    string `json:"lambda_name"`
	Type          string `json:"type"`
	Token         string `json:"token"`
	Message       []byte `json:"message"`
	LambdaVersion int    `json:"lambda_version"`
}
