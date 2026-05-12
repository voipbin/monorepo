package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	log "github.com/sirupsen/logrus"
)

// request is structurally compatible with monorepo/bin-common-handler/models/sock.Request.
// Method is string here (vs sock.RequestMethod typedef) — wire format is identical.
// RequestID is included for log correlation across the call chain.
type request struct {
	URI       string          `json:"uri"`
	Method    string          `json:"method"` // valid values: POST, GET, PUT, DELETE
	Publisher string          `json:"publisher"`
	DataType  string          `json:"data_type"`
	Data      json.RawMessage `json:"data,omitempty"`
	RequestID string          `json:"request_id,omitempty"`
}

// response mirrors monorepo/bin-common-handler/models/sock.Response
type response struct {
	StatusCode int             `json:"status_code"`
	DataType   string          `json:"data_type"`
	Data       json.RawMessage `json:"data,omitempty"`
}

func main() {
	rabbitAddr := flag.String("rabbit_addr", "", "RabbitMQ address (amqp://...)")
	queue      := flag.String("queue", "", "Target queue name")
	uri        := flag.String("uri", "", "Request URI (e.g. /v1/numbers/renew)")
	method     := flag.String("method", "POST", "Request method (POST, GET, ...)")
	dataType   := flag.String("data_type", "application/json", "Content type")
	data       := flag.String("data", "", "Request body as JSON string")
	timeoutMs  := flag.Int("timeout", 5000, "Timeout in milliseconds")
	delayMs    := flag.Int("delay", 0, "Delay before sending in milliseconds")
	flag.Parse()

	if *rabbitAddr == "" || *queue == "" || *uri == "" {
		fmt.Fprintln(os.Stderr, "required: -rabbit_addr, -queue, -uri")
		flag.Usage()
		os.Exit(1)
	}

	if *data != "" && !json.Valid([]byte(*data)) {
		fmt.Fprintf(os.Stderr, "invalid JSON in -data flag: %s\n", *data)
		os.Exit(1)
	}

	if *delayMs > 0 {
		log.Infof("Delaying %d ms before sending", *delayMs)
		time.Sleep(time.Duration(*delayMs) * time.Millisecond)
	}

	if err := run(*rabbitAddr, *queue, *uri, *method, *dataType, *data, *timeoutMs); err != nil {
		log.Errorf("Failed: %v", err)
		os.Exit(1)
	}
}

func buildRequest(uri, method, dataType, data string) *request {
	req := request{
		URI:       uri,
		Method:    method,
		Publisher: "bin-trigger-sender",
		DataType:  dataType,
		RequestID: "bin-trigger-sender-cronjob",
	}
	if data != "" {
		req.Data = json.RawMessage(data)
	}
	return &req
}

func run(rabbitAddr, queue, uri, method, dataType, data string, timeoutMs int) error {
	conn, err := amqp.Dial(rabbitAddr)
	if err != nil {
		return fmt.Errorf("dial RabbitMQ: %w", err)
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			log.Warnf("close connection: %v", closeErr)
		}
	}()

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	defer func() {
		if closeErr := ch.Close(); closeErr != nil {
			log.Warnf("close channel: %v", closeErr)
		}
	}()

	replyQ, err := ch.QueueDeclare("", false, true, true, false, nil)
	if err != nil {
		return fmt.Errorf("declare reply queue: %w", err)
	}

	msgs, err := ch.Consume(replyQ.Name, "", true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume reply queue: %w", err)
	}

	req := buildRequest(uri, method, dataType, data)

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	err = ch.PublishWithContext(ctx, "", queue, false, false,
		amqp.Publishing{
			ContentType: "application/json",
			ReplyTo:     replyQ.Name,
			Body:        body,
		})
	if err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	log.Infof("Sent request to queue=%s uri=%s method=%s", queue, uri, method)

	select {
	case <-ctx.Done():
		return fmt.Errorf("timeout waiting for response after %d ms", timeoutMs)
	case msg := <-msgs:
		var res response
		if err := json.Unmarshal(msg.Body, &res); err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
		log.Infof("Response: status=%d data=%s", res.StatusCode, res.Data)
		if res.StatusCode < 200 || res.StatusCode >= 300 {
			return fmt.Errorf("non-2xx response: %d", res.StatusCode)
		}
		log.Infof("Request completed successfully")
		return nil
	}
}
