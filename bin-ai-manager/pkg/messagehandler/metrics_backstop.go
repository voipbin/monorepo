package messagehandler

import "github.com/prometheus/client_golang/prometheus"

// promBackstopReplyTotal counts pipecatcall_terminated backstop attempts in
// EventPMPipecatcallTerminated by their final result. Labels:
//   - sent                   — backstop reply persisted and sent
//   - failed                 — persistence failed before send
//   - send_failed            — persisted but ConversationV1MessageSend errored
//   - skipped_seen           — assistant reply already exists (short-circuit)
//   - skipped_voice          : AIcall reference is neither a conversation nor a contact_case
//   - skipped_terminated     — AIcall is already in terminated status
//   - skipped_not_aicall     — pipecatcall reference is not an AICall
//   - timeout_notice_created : contact_case turn ended unanswered; a timeout notice was recorded (VOIP-1633)
//   - timeout_notice_failed  : the timeout notice could not be recorded
//   - timeout_notice_skipped : contact_case terminated event with no notice due (answered, listen turn,
//     explicit end, or a notice already exists)
//
// The counter lives in this package (not aicallhandler) because aicallhandler
// imports messagehandler; placing it in aicallhandler would create a circular
// import. The fully-qualified name (`ai_manager_aicall_backstop_reply_total`)
// is preserved by sharing `metricsNamespace = "ai_manager"`.
var (
	promBackstopReplyTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "aicall_backstop_reply_total",
			Help:      "Counter of pipecatcall_terminated backstop attempts in messagehandler.EventPMPipecatcallTerminated.",
		},
		[]string{"result"},
	)
)

func init() {
	prometheus.MustRegister(promBackstopReplyTotal)
}
