package carriertrackhandler

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"
	"unicode"
)

// Status is the normalized state of a package.
type Status string

// The normalized states.
const (
	StatusLabelCreated       Status = "label_created"
	StatusInTransit          Status = "in_transit"
	StatusOutForDelivery     Status = "out_for_delivery"
	StatusDelivered          Status = "delivered"
	StatusException          Status = "exception"
	StatusReturned           Status = "returned"
	StatusUnknown            Status = "unknown"
	maxEvents                       = 8
	maxDisplayEvents                = 5
	maxDisplayFields                = 12
	maxTextLen                      = 200
	maxLocationLen                  = 120
	maxServiceLen                   = 80
	maxStatusCodeLen                = 40
	maxDisplayValueLen              = 500
	maxDisplayLabelLen              = 50
	maxDisplayTitleLen              = 200
	maxDisplayDescriptionLen        = 1000
	redactedText                    = "[redacted]"

	// resultNote is added to every result: a tracking number taken from a
	// phone call may have been misheard.
	resultNote = "The tracking number may have been recognized from speech. Ask the agent to confirm it before relying on this result."
)

var validStatuses = map[Status]bool{
	StatusLabelCreated: true, StatusInTransit: true, StatusOutForDelivery: true, StatusDelivered: true,
	StatusException: true, StatusReturned: true, StatusUnknown: true,
}

// Location is where an event happened, as text only.
type Location struct {
	Text string `json:"text"`
}

// Event is one scan of the package.
type Event struct {
	Time        *string   `json:"time"`
	TimeLocal   *string   `json:"time_local"`
	Description string    `json:"description"`
	Location    *Location `json:"location"`
}

// EstimatedDelivery holds what the carrier gave: a date, a window, or none.
type EstimatedDelivery struct {
	Date *string `json:"date"`
	From *string `json:"from"`
	To   *string `json:"to"`
}

// DisplayField is one label and value of the card.
type DisplayField struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// Display has the shape of an info card, so the model can pass it on as it is.
type Display struct {
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Fields      []DisplayField `json:"fields"`
}

// Result is the normalized answer of a lookup. It is the allow list of what
// the model, the stored message and the customer's webhook can receive: a
// field that is not here cannot leave the adapter. Recipient and sender data,
// addresses, signatures, photos and payment data have no field.
type Result struct {
	Carrier           string            `json:"carrier"`
	TrackingNumber    string            `json:"tracking_number"`
	Status            Status            `json:"status"`
	StatusText        string            `json:"status_text"`
	DeliveredAt       *string           `json:"delivered_at"`
	TimeZoneKnown     bool              `json:"time_zone_known"`
	EstimatedDelivery EstimatedDelivery `json:"estimated_delivery"`
	LastEvent         *Event            `json:"last_event"`
	Events            []Event           `json:"events"`
	Service           *string           `json:"service"`
	CarrierStatusCode *string           `json:"carrier_status_code"`
	EventsTruncated   bool              `json:"events_truncated"`
	Note              string            `json:"note"`
	FetchedAt         string            `json:"fetched_at"`
	Cached            bool              `json:"cached"`
	Display           Display           `json:"display"`
}

var (
	emailPattern = regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`)
	phonePattern = regexp.MustCompile(`\+?\d[\d\s().\-]{7,}\d`)
)

// sanitizeText makes carrier free text safe to hand on: control characters and
// line breaks become spaces, e-mail addresses and phone numbers are masked,
// and the result is cut to limit runes. The text stays data; nothing here tries
// to detect instructions.
func sanitizeText(s string, limit int) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, s)
	s = emailPattern.ReplaceAllString(s, redactedText)
	s = phonePattern.ReplaceAllString(s, redactedText)
	s = strings.Join(strings.Fields(s), " ")

	runes := []rune(s)
	if len(runes) > limit {
		runes = runes[:limit]
	}
	return strings.TrimSpace(string(runes))
}

func sanitizePtr(p *string, limit int) *string {
	if p == nil {
		return nil
	}
	s := sanitizeText(*p, limit)
	if s == "" {
		return nil
	}
	return &s
}

// finalize applies every cap, fills the fixed fields and builds the card. It
// is the one place a Result becomes safe to serialize.
func (r *Result) finalize(carrier, number string, now time.Time) {
	r.Carrier = carrier
	r.TrackingNumber = number
	r.Note = resultNote
	r.FetchedAt = now.UTC().Format(time.RFC3339)
	r.Cached = false

	if !validStatuses[r.Status] {
		r.Status = StatusUnknown
	}
	r.StatusText = sanitizeText(r.StatusText, maxTextLen)
	r.Service = sanitizePtr(r.Service, maxServiceLen)
	r.CarrierStatusCode = sanitizePtr(r.CarrierStatusCode, maxStatusCodeLen)

	if len(r.Events) > maxEvents {
		r.Events = r.Events[:maxEvents]
		r.EventsTruncated = true
	}
	events := make([]Event, 0, len(r.Events))
	for _, e := range r.Events {
		events = append(events, cleanEvent(e))
	}
	r.Events = events

	if r.LastEvent != nil {
		e := cleanEvent(*r.LastEvent)
		r.LastEvent = &e
	} else if len(r.Events) > 0 {
		e := r.Events[0]
		r.LastEvent = &e
	}

	r.Display = r.buildDisplay()
}

func cleanEvent(e Event) Event {
	out := Event{
		Time:        e.Time,
		TimeLocal:   sanitizePtr(e.TimeLocal, maxServiceLen),
		Description: sanitizeText(e.Description, maxTextLen),
	}
	if e.Location != nil {
		if text := sanitizeText(e.Location.Text, maxLocationLen); text != "" {
			out.Location = &Location{Text: text}
		}
	}
	return out
}

func (r *Result) buildDisplay() Display {
	name := displayName(r.Carrier)
	fields := []DisplayField{}
	add := func(label, value string) {
		value = sanitizeText(value, maxDisplayValueLen)
		if value == "" || len(fields) >= maxDisplayFields {
			return
		}
		fields = append(fields, DisplayField{Label: sanitizeText(label, maxDisplayLabelLen), Value: value})
	}

	add("Carrier", name)
	add("Tracking number", r.TrackingNumber)
	add("Status", string(r.Status))
	add("Status detail", r.StatusText)
	switch {
	case r.DeliveredAt != nil:
		add("Delivered", *r.DeliveredAt)
	case r.EstimatedDelivery.Date != nil:
		add("Estimated delivery", *r.EstimatedDelivery.Date)
	case r.EstimatedDelivery.From != nil && r.EstimatedDelivery.To != nil:
		add("Estimated delivery", *r.EstimatedDelivery.From+" to "+*r.EstimatedDelivery.To)
	}
	if r.Service != nil {
		add("Service", *r.Service)
	}
	if r.LastEvent != nil && r.LastEvent.Time != nil {
		add("Last update", *r.LastEvent.Time)
	}

	for i, e := range r.Events {
		if i >= maxDisplayEvents {
			break
		}
		label := "Event"
		if e.Time != nil {
			label = *e.Time
		} else if e.TimeLocal != nil {
			label = *e.TimeLocal
		}
		value := e.Description
		if e.Location != nil {
			value += " (" + e.Location.Text + ")"
		}
		add(label, value)
	}

	return Display{
		Title:       sanitizeText(name+" package "+r.TrackingNumber, maxDisplayTitleLen),
		Description: sanitizeText(r.StatusText, maxDisplayDescriptionLen),
		Fields:      fields,
	}
}

// marshal serializes the result.
func (r *Result) marshal() (string, error) {
	b, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
