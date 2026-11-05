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
	maxTimeLen                      = 64
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
	// A phone number is masked when it is written with separators or a plus
	// sign. A plain run of digits is left alone: tracking numbers and dates
	// look the same and the card needs them.
	phonePattern = regexp.MustCompile(`\+\d{8,15}|(?:\+\d{1,3}[\s.\-]?)?(?:\(\d{2,4}\)[\s.\-]?|\d{2,4}[\s.\-])\d{3,4}[\s.\-]\d{3,4}|\+\d{1,3}[\s.\-]\d{2}[\s.\-]\d{5,8}`)
)

// sanitizeText makes carrier free text safe to hand on: control characters and
// line breaks become spaces, e-mail addresses and phone numbers are masked,
// and the result is cut to limit runes. The text stays data; nothing here tries
// to detect instructions.
func sanitizeText(s string, limit int) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
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

// cleanStructured bounds a field that holds a date, a time or an identifier.
// Control characters become spaces and the value is cut to limit runes; nothing
// is masked, because masking would damage the value.
func cleanStructured(s string, limit int) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return ' '
		}
		return r
	}, s)
	runes := []rune(strings.Join(strings.Fields(s), " "))
	if len(runes) > limit {
		runes = runes[:limit]
	}
	return strings.TrimSpace(string(runes))
}

func cleanStructuredPtr(p *string) *string {
	if p == nil {
		return nil
	}
	s := cleanStructured(*p, maxTimeLen)
	if s == "" {
		return nil
	}
	return &s
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
	r.DeliveredAt = cleanStructuredPtr(r.DeliveredAt)
	r.EstimatedDelivery = EstimatedDelivery{
		Date: cleanStructuredPtr(r.EstimatedDelivery.Date),
		From: cleanStructuredPtr(r.EstimatedDelivery.From),
		To:   cleanStructuredPtr(r.EstimatedDelivery.To),
	}

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
		Time:        cleanStructuredPtr(e.Time),
		TimeLocal:   cleanStructuredPtr(e.TimeLocal),
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
	addValue := func(label, value string) {
		if value == "" || len(fields) >= maxDisplayFields {
			return
		}
		fields = append(fields, DisplayField{Label: cleanStructured(label, maxDisplayLabelLen), Value: value})
	}
	add := func(label, value string) {
		value = sanitizeText(value, maxDisplayValueLen)
		if value == "" || len(fields) >= maxDisplayFields {
			return
		}
		fields = append(fields, DisplayField{Label: cleanStructured(label, maxDisplayLabelLen), Value: value})
	}

	add("Carrier", name)
	addValue("Tracking number", cleanStructured(r.TrackingNumber, maxTimeLen))
	add("Status", string(r.Status))
	add("Status detail", r.StatusText)
	switch {
	case r.DeliveredAt != nil:
		addValue("Delivered", *r.DeliveredAt)
	case r.EstimatedDelivery.Date != nil:
		addValue("Estimated delivery", *r.EstimatedDelivery.Date)
	case r.EstimatedDelivery.From != nil && r.EstimatedDelivery.To != nil:
		addValue("Estimated delivery", *r.EstimatedDelivery.From+" to "+*r.EstimatedDelivery.To)
	}
	if r.Service != nil {
		add("Service", *r.Service)
	}
	if r.LastEvent != nil && r.LastEvent.Time != nil {
		addValue("Last update", *r.LastEvent.Time)
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
		label = cleanStructured(label, maxDisplayLabelLen)
		value := e.Description
		if e.Location != nil {
			value += " (" + e.Location.Text + ")"
		}
		add(label, value)
	}

	return Display{
		Title:       cleanStructured(name+" package "+r.TrackingNumber, maxDisplayTitleLen),
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
