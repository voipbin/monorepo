package siphandler

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// HealthCheckResult holds the result of a SIP OPTIONS health check.
type HealthCheckResult struct {
	Healthy      bool
	ResponseCode string // SIP response code e.g. "200", "404", or "timeout"
}

// SIPChecker is a function type for sending SIP OPTIONS health checks.
// Using a function type allows injection of test stubs without a full interface.
type SIPChecker func(ctx context.Context, hostname string, timeout time.Duration) (*HealthCheckResult, error)

// SendOptionsCheck sends SIP OPTIONS to hostname:5060 via UDP.
// Returns healthy=true for any SIP response; healthy=false on timeout/error.
// The error return is always nil — errors are converted to unhealthy results.
func SendOptionsCheck(ctx context.Context, hostname string, timeout time.Duration) (*HealthCheckResult, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "SendOptionsCheck",
		"hostname": hostname,
		"timeout":  timeout,
	})

	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:5060", hostname))
	if err != nil {
		log.Debugf("Could not resolve UDP address. hostname: %s, err: %v", hostname, err)
		return &HealthCheckResult{Healthy: false, ResponseCode: "timeout"}, nil
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		log.Debugf("Could not dial UDP. hostname: %s, err: %v", hostname, err)
		return &HealthCheckResult{Healthy: false, ResponseCode: "timeout"}, nil
	}
	defer conn.Close() //nolint:errcheck

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	localIP := localAddr.IP.String()
	localPort := localAddr.Port

	//nolint:gosec // non-crypto random is fine for SIP branch/call-id generation
	callID := fmt.Sprintf("%x", rand.Int63())
	//nolint:gosec
	tag := fmt.Sprintf("%x", rand.Int31())
	//nolint:gosec
	branch := fmt.Sprintf("z9hG4bK%x", rand.Int31())

	msg := fmt.Sprintf(
		"OPTIONS sip:healthcheck@%s SIP/2.0\r\n"+
			"Via: SIP/2.0/UDP %s:%d;branch=%s\r\n"+
			"From: <sip:healthcheck@%s>;tag=%s\r\n"+
			"To: <sip:healthcheck@%s>\r\n"+
			"Call-ID: %s@%s\r\n"+
			"CSeq: 1 OPTIONS\r\n"+
			"Max-Forwards: 70\r\n"+
			"Contact: <sip:healthcheck@%s:%d>\r\n"+
			"Content-Length: 0\r\n\r\n",
		hostname,
		localIP, localPort, branch,
		localIP, tag,
		hostname,
		callID, localIP,
		localIP, localPort,
	)

	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		log.Debugf("Could not set deadline. err: %v", err)
		return &HealthCheckResult{Healthy: false, ResponseCode: "timeout"}, nil
	}

	if _, err := conn.Write([]byte(msg)); err != nil {
		log.Debugf("Could not send SIP OPTIONS. hostname: %s, err: %v", hostname, err)
		return &HealthCheckResult{Healthy: false, ResponseCode: "timeout"}, nil
	}

	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		// timeout or network error → unhealthy
		log.Debugf("No SIP response received (timeout or error). hostname: %s, err: %v", hostname, err)
		return &HealthCheckResult{Healthy: false, ResponseCode: "timeout"}, nil
	}

	// Parse first line: "SIP/2.0 <code> <reason>"
	firstLine := strings.SplitN(string(buf[:n]), "\r\n", 2)[0]
	parts := strings.Fields(firstLine)
	responseCode := "unknown"
	if len(parts) >= 2 {
		responseCode = parts[1]
	}

	log.Debugf("SIP response received. hostname: %s, code: %s", hostname, responseCode)
	return &HealthCheckResult{Healthy: true, ResponseCode: responseCode}, nil
}
