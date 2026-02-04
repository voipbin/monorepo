# Contact API Endpoints Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Expose contact CRUD, lookup, and nested resource endpoints through the public REST API (bin-api-manager).

**Architecture:** The contact-manager service already handles all contact operations via RabbitMQ RPC. We need to add: (1) OpenAPI definitions for the endpoints, (2) RequestHandler methods in bin-common-handler to make RPC calls, (3) ServiceHandler methods in bin-api-manager to call the RequestHandler with permission checks, (4) Server handlers in bin-api-manager to expose HTTP endpoints.

**Tech Stack:** Go, OpenAPI 3.0, RabbitMQ RPC, Gin HTTP framework

---

## Task 1: Add OpenAPI Schema Definitions

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml`

**Step 1: Add Contact schemas to openapi.yaml**

Add the following schemas to the `components/schemas` section of `openapi.yaml`:

```yaml
    ContactManagerContact:
      type: object
      properties:
        id:
          type: string
          format: uuid
        customer_id:
          type: string
          format: uuid
        first_name:
          type: string
        last_name:
          type: string
        display_name:
          type: string
        company:
          type: string
        job_title:
          type: string
        source:
          type: string
          enum: [manual, import, api, sync]
        external_id:
          type: string
        phone_numbers:
          type: array
          items:
            $ref: '#/components/schemas/ContactManagerPhoneNumber'
        emails:
          type: array
          items:
            $ref: '#/components/schemas/ContactManagerEmail'
        tag_ids:
          type: array
          items:
            type: string
            format: uuid
        tm_create:
          type: string
          format: date-time
        tm_update:
          type: string
          format: date-time
        tm_delete:
          type: string
          format: date-time

    ContactManagerPhoneNumber:
      type: object
      properties:
        id:
          type: string
          format: uuid
        number:
          type: string
        number_e164:
          type: string
        type:
          type: string
          enum: [mobile, work, home, fax, other]
        is_primary:
          type: boolean
        tm_create:
          type: string
          format: date-time

    ContactManagerEmail:
      type: object
      properties:
        id:
          type: string
          format: uuid
        address:
          type: string
          format: email
        type:
          type: string
          enum: [work, personal, other]
        is_primary:
          type: boolean
        tm_create:
          type: string
          format: date-time
```

**Step 2: Commit**

```bash
git add bin-openapi-manager/openapi/openapi.yaml
git commit -m "NOJIRA-Add-contact-api-endpoints

- bin-openapi-manager: Add ContactManagerContact, ContactManagerPhoneNumber, ContactManagerEmail schemas"
```

---

## Task 2: Create OpenAPI Path Definitions - Core CRUD

**Files:**
- Create: `bin-openapi-manager/openapi/paths/contacts/main.yaml`
- Create: `bin-openapi-manager/openapi/paths/contacts/id.yaml`

**Step 1: Create contacts/main.yaml**

```yaml
get:
  summary: List contacts
  description: Get contacts for the customer.
  tags:
    - Contact
  parameters:
    - $ref: '#/components/parameters/PageSize'
    - $ref: '#/components/parameters/PageToken'
  responses:
    '200':
      description: Successful response.
      content:
        application/json:
          schema:
            allOf:
              - $ref: '#/components/schemas/CommonPagination'
              - type: object
                properties:
                  result:
                    type: array
                    items:
                      $ref: '#/components/schemas/ContactManagerContact'

post:
  summary: Create a new contact
  description: Create a new contact for the customer.
  tags:
    - Contact
  requestBody:
    description: Request body to create a new contact.
    required: true
    content:
      application/json:
        schema:
          type: object
          properties:
            first_name:
              type: string
            last_name:
              type: string
            display_name:
              type: string
            company:
              type: string
            job_title:
              type: string
            source:
              type: string
              enum: [manual, import, api, sync]
            external_id:
              type: string
            notes:
              type: string
            phone_numbers:
              type: array
              items:
                type: object
                properties:
                  number:
                    type: string
                  type:
                    type: string
                    enum: [mobile, work, home, fax, other]
                  is_primary:
                    type: boolean
            emails:
              type: array
              items:
                type: object
                properties:
                  address:
                    type: string
                    format: email
                  type:
                    type: string
                    enum: [work, personal, other]
                  is_primary:
                    type: boolean
            tag_ids:
              type: array
              items:
                type: string
                format: uuid
  responses:
    '201':
      description: Contact created successfully.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/ContactManagerContact'
    '400':
      description: Invalid input.
```

**Step 2: Create contacts/id.yaml**

```yaml
get:
  summary: Get the contact
  description: Get the contact of the given ID.
  tags:
    - Contact
  parameters:
    - name: id
      in: path
      description: The ID of the contact.
      required: true
      schema:
        type: string
  responses:
    '200':
      description: Successful response.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/ContactManagerContact'

put:
  summary: Update a contact
  description: Update a contact and return updated details.
  tags:
    - Contact
  parameters:
    - name: id
      in: path
      description: The ID of the contact.
      required: true
      schema:
        type: string
  requestBody:
    required: true
    content:
      application/json:
        schema:
          type: object
          properties:
            first_name:
              type: string
            last_name:
              type: string
            display_name:
              type: string
            company:
              type: string
            job_title:
              type: string
            external_id:
              type: string
            notes:
              type: string
  responses:
    '200':
      description: Successful response.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/ContactManagerContact'

delete:
  summary: Delete the contact
  description: Delete the contact of the given ID.
  tags:
    - Contact
  parameters:
    - name: id
      in: path
      description: The ID of the contact.
      required: true
      schema:
        type: string
  responses:
    '200':
      description: Successful response.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/ContactManagerContact'
```

**Step 3: Commit**

```bash
git add bin-openapi-manager/openapi/paths/contacts/
git commit -m "NOJIRA-Add-contact-api-endpoints

- bin-openapi-manager: Add OpenAPI path definitions for contacts CRUD endpoints"
```

---

## Task 3: Create OpenAPI Path Definitions - Lookup and Nested Resources

**Files:**
- Create: `bin-openapi-manager/openapi/paths/contacts/lookup.yaml`
- Create: `bin-openapi-manager/openapi/paths/contacts/id_phonenumbers.yaml`
- Create: `bin-openapi-manager/openapi/paths/contacts/id_phonenumbers_id.yaml`
- Create: `bin-openapi-manager/openapi/paths/contacts/id_emails.yaml`
- Create: `bin-openapi-manager/openapi/paths/contacts/id_emails_id.yaml`
- Create: `bin-openapi-manager/openapi/paths/contacts/id_tags.yaml`
- Create: `bin-openapi-manager/openapi/paths/contacts/id_tags_id.yaml`

**Step 1: Create contacts/lookup.yaml**

```yaml
get:
  summary: Lookup contact
  description: Find a contact by phone number or email.
  tags:
    - Contact
  parameters:
    - name: phone
      in: query
      description: Phone number in E.164 format to lookup.
      required: false
      schema:
        type: string
    - name: email
      in: query
      description: Email address to lookup.
      required: false
      schema:
        type: string
  responses:
    '200':
      description: Contact found.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/ContactManagerContact'
    '404':
      description: Contact not found.
```

**Step 2: Create contacts/id_phonenumbers.yaml**

```yaml
post:
  summary: Add phone number to contact
  description: Add a new phone number to the contact.
  tags:
    - Contact
  parameters:
    - name: id
      in: path
      description: The ID of the contact.
      required: true
      schema:
        type: string
  requestBody:
    required: true
    content:
      application/json:
        schema:
          type: object
          required:
            - number
          properties:
            number:
              type: string
            type:
              type: string
              enum: [mobile, work, home, fax, other]
            is_primary:
              type: boolean
  responses:
    '200':
      description: Phone number added successfully.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/ContactManagerContact'
```

**Step 3: Create contacts/id_phonenumbers_id.yaml**

```yaml
delete:
  summary: Remove phone number from contact
  description: Remove a phone number from the contact.
  tags:
    - Contact
  parameters:
    - name: id
      in: path
      description: The ID of the contact.
      required: true
      schema:
        type: string
    - name: phone_number_id
      in: path
      description: The ID of the phone number.
      required: true
      schema:
        type: string
  responses:
    '200':
      description: Phone number removed successfully.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/ContactManagerContact'
```

**Step 4: Create contacts/id_emails.yaml**

```yaml
post:
  summary: Add email to contact
  description: Add a new email address to the contact.
  tags:
    - Contact
  parameters:
    - name: id
      in: path
      description: The ID of the contact.
      required: true
      schema:
        type: string
  requestBody:
    required: true
    content:
      application/json:
        schema:
          type: object
          required:
            - address
          properties:
            address:
              type: string
              format: email
            type:
              type: string
              enum: [work, personal, other]
            is_primary:
              type: boolean
  responses:
    '200':
      description: Email added successfully.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/ContactManagerContact'
```

**Step 5: Create contacts/id_emails_id.yaml**

```yaml
delete:
  summary: Remove email from contact
  description: Remove an email address from the contact.
  tags:
    - Contact
  parameters:
    - name: id
      in: path
      description: The ID of the contact.
      required: true
      schema:
        type: string
    - name: email_id
      in: path
      description: The ID of the email.
      required: true
      schema:
        type: string
  responses:
    '200':
      description: Email removed successfully.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/ContactManagerContact'
```

**Step 6: Create contacts/id_tags.yaml**

```yaml
post:
  summary: Add tag to contact
  description: Add a tag to the contact.
  tags:
    - Contact
  parameters:
    - name: id
      in: path
      description: The ID of the contact.
      required: true
      schema:
        type: string
  requestBody:
    required: true
    content:
      application/json:
        schema:
          type: object
          required:
            - tag_id
          properties:
            tag_id:
              type: string
              format: uuid
  responses:
    '200':
      description: Tag added successfully.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/ContactManagerContact'
```

**Step 7: Create contacts/id_tags_id.yaml**

```yaml
delete:
  summary: Remove tag from contact
  description: Remove a tag from the contact.
  tags:
    - Contact
  parameters:
    - name: id
      in: path
      description: The ID of the contact.
      required: true
      schema:
        type: string
    - name: tag_id
      in: path
      description: The ID of the tag.
      required: true
      schema:
        type: string
  responses:
    '200':
      description: Tag removed successfully.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/ContactManagerContact'
```

**Step 8: Commit**

```bash
git add bin-openapi-manager/openapi/paths/contacts/
git commit -m "NOJIRA-Add-contact-api-endpoints

- bin-openapi-manager: Add OpenAPI path definitions for contact lookup and nested resources"
```

---

## Task 4: Register Contact Paths in OpenAPI

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml`

**Step 1: Add path references to openapi.yaml**

Add these entries to the `paths` section (in alphabetical order with other paths):

```yaml
  /v1/contacts:
    $ref: './paths/contacts/main.yaml'
  /v1/contacts/lookup:
    $ref: './paths/contacts/lookup.yaml'
  /v1/contacts/{id}:
    $ref: './paths/contacts/id.yaml'
  /v1/contacts/{id}/phone-numbers:
    $ref: './paths/contacts/id_phonenumbers.yaml'
  /v1/contacts/{id}/phone-numbers/{phone_number_id}:
    $ref: './paths/contacts/id_phonenumbers_id.yaml'
  /v1/contacts/{id}/emails:
    $ref: './paths/contacts/id_emails.yaml'
  /v1/contacts/{id}/emails/{email_id}:
    $ref: './paths/contacts/id_emails_id.yaml'
  /v1/contacts/{id}/tags:
    $ref: './paths/contacts/id_tags.yaml'
  /v1/contacts/{id}/tags/{tag_id}:
    $ref: './paths/contacts/id_tags_id.yaml'
```

**Step 2: Run verification workflow for bin-openapi-manager**

```bash
cd bin-openapi-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 3: Commit**

```bash
git add bin-openapi-manager/
git commit -m "NOJIRA-Add-contact-api-endpoints

- bin-openapi-manager: Register contact paths in openapi.yaml"
```

---

## Task 5: Create Request Handler - Contact CRUD

**Files:**
- Create: `bin-common-handler/pkg/requesthandler/contact_contacts.go`

**Step 1: Create contact_contacts.go**

```go
package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	cmcontact "monorepo/bin-contact-manager/models/contact"
	cmrequest "monorepo/bin-contact-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/sock"
)

// ContactV1ContactCreate sends a request to contact-manager to create a contact.
func (r *requestHandler) ContactV1ContactCreate(
	ctx context.Context,
	customerID uuid.UUID,
	firstName string,
	lastName string,
	displayName string,
	company string,
	jobTitle string,
	source string,
	externalID string,
	notes string,
	phoneNumbers []cmrequest.PhoneNumberCreate,
	emails []cmrequest.EmailCreate,
	tagIDs []uuid.UUID,
) (*cmcontact.Contact, error) {
	uri := "/v1/contacts"

	data := &cmrequest.ContactCreate{
		CustomerID:   customerID,
		FirstName:    firstName,
		LastName:     lastName,
		DisplayName:  displayName,
		Company:      company,
		JobTitle:     jobTitle,
		Source:       source,
		ExternalID:   externalID,
		Notes:        notes,
		PhoneNumbers: phoneNumbers,
		Emails:       emails,
		TagIDs:       tagIDs,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodPost, "contact/contacts", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cmcontact.Contact
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ContactV1ContactGet sends a request to contact-manager to get a contact.
func (r *requestHandler) ContactV1ContactGet(ctx context.Context, contactID uuid.UUID) (*cmcontact.Contact, error) {
	uri := fmt.Sprintf("/v1/contacts/%s", contactID)

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodGet, "contact/contacts/<contact-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res cmcontact.Contact
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ContactV1ContactList sends a request to contact-manager to list contacts.
func (r *requestHandler) ContactV1ContactList(ctx context.Context, pageToken string, pageSize uint64, filters map[cmcontact.Field]any) ([]cmcontact.Contact, error) {
	uri := fmt.Sprintf("/v1/contacts?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodGet, "contact/contacts", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []cmcontact.Contact
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// ContactV1ContactUpdate sends a request to contact-manager to update a contact.
func (r *requestHandler) ContactV1ContactUpdate(
	ctx context.Context,
	contactID uuid.UUID,
	firstName *string,
	lastName *string,
	displayName *string,
	company *string,
	jobTitle *string,
	externalID *string,
	notes *string,
) (*cmcontact.Contact, error) {
	uri := fmt.Sprintf("/v1/contacts/%s", contactID)

	data := &cmrequest.ContactUpdate{
		FirstName:   firstName,
		LastName:    lastName,
		DisplayName: displayName,
		Company:     company,
		JobTitle:    jobTitle,
		ExternalID:  externalID,
		Notes:       notes,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodPut, "contact/contacts/<contact-id>", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cmcontact.Contact
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ContactV1ContactDelete sends a request to contact-manager to delete a contact.
func (r *requestHandler) ContactV1ContactDelete(ctx context.Context, contactID uuid.UUID) (*cmcontact.Contact, error) {
	uri := fmt.Sprintf("/v1/contacts/%s", contactID)

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodDelete, "contact/contacts/<contact-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res cmcontact.Contact
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ContactV1ContactLookup sends a request to contact-manager to lookup a contact by phone or email.
func (r *requestHandler) ContactV1ContactLookup(ctx context.Context, customerID uuid.UUID, phoneE164 string, email string) (*cmcontact.Contact, error) {
	uri := fmt.Sprintf("/v1/contacts/lookup?customer_id=%s", customerID)
	if phoneE164 != "" {
		uri = fmt.Sprintf("%s&phone_e164=%s", uri, url.QueryEscape(phoneE164))
	}
	if email != "" {
		uri = fmt.Sprintf("%s&email=%s", uri, url.QueryEscape(email))
	}

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodGet, "contact/contacts/lookup", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res cmcontact.Contact
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
```

**Step 2: Commit**

```bash
git add bin-common-handler/pkg/requesthandler/contact_contacts.go
git commit -m "NOJIRA-Add-contact-api-endpoints

- bin-common-handler: Add request handler methods for contact CRUD and lookup"
```

---

## Task 6: Create Request Handler - Nested Resources

**Files:**
- Create: `bin-common-handler/pkg/requesthandler/contact_phonenumbers.go`
- Create: `bin-common-handler/pkg/requesthandler/contact_emails.go`
- Create: `bin-common-handler/pkg/requesthandler/contact_tags.go`

**Step 1: Create contact_phonenumbers.go**

```go
package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	cmcontact "monorepo/bin-contact-manager/models/contact"
	cmrequest "monorepo/bin-contact-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/sock"
)

// ContactV1PhoneNumberCreate sends a request to contact-manager to add a phone number to a contact.
func (r *requestHandler) ContactV1PhoneNumberCreate(
	ctx context.Context,
	contactID uuid.UUID,
	number string,
	numberE164 string,
	phoneType string,
	isPrimary bool,
) (*cmcontact.Contact, error) {
	uri := fmt.Sprintf("/v1/contacts/%s/phone-numbers", contactID)

	data := &cmrequest.PhoneNumberCreate{
		Number:     number,
		NumberE164: numberE164,
		Type:       phoneType,
		IsPrimary:  isPrimary,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodPost, "contact/contacts/<contact-id>/phone-numbers", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cmcontact.Contact
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ContactV1PhoneNumberDelete sends a request to contact-manager to remove a phone number from a contact.
func (r *requestHandler) ContactV1PhoneNumberDelete(ctx context.Context, contactID uuid.UUID, phoneNumberID uuid.UUID) (*cmcontact.Contact, error) {
	uri := fmt.Sprintf("/v1/contacts/%s/phone-numbers/%s", contactID, phoneNumberID)

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodDelete, "contact/contacts/<contact-id>/phone-numbers/<phone-number-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res cmcontact.Contact
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
```

**Step 2: Create contact_emails.go**

```go
package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	cmcontact "monorepo/bin-contact-manager/models/contact"
	cmrequest "monorepo/bin-contact-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/sock"
)

// ContactV1EmailCreate sends a request to contact-manager to add an email to a contact.
func (r *requestHandler) ContactV1EmailCreate(
	ctx context.Context,
	contactID uuid.UUID,
	address string,
	emailType string,
	isPrimary bool,
) (*cmcontact.Contact, error) {
	uri := fmt.Sprintf("/v1/contacts/%s/emails", contactID)

	data := &cmrequest.EmailCreate{
		Address:   address,
		Type:      emailType,
		IsPrimary: isPrimary,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodPost, "contact/contacts/<contact-id>/emails", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cmcontact.Contact
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ContactV1EmailDelete sends a request to contact-manager to remove an email from a contact.
func (r *requestHandler) ContactV1EmailDelete(ctx context.Context, contactID uuid.UUID, emailID uuid.UUID) (*cmcontact.Contact, error) {
	uri := fmt.Sprintf("/v1/contacts/%s/emails/%s", contactID, emailID)

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodDelete, "contact/contacts/<contact-id>/emails/<email-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res cmcontact.Contact
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
```

**Step 3: Create contact_tags.go**

```go
package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	cmcontact "monorepo/bin-contact-manager/models/contact"
	cmrequest "monorepo/bin-contact-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/sock"
)

// ContactV1TagAdd sends a request to contact-manager to add a tag to a contact.
func (r *requestHandler) ContactV1TagAdd(ctx context.Context, contactID uuid.UUID, tagID uuid.UUID) (*cmcontact.Contact, error) {
	uri := fmt.Sprintf("/v1/contacts/%s/tags", contactID)

	data := &cmrequest.TagAssignment{
		TagID: tagID,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodPost, "contact/contacts/<contact-id>/tags", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cmcontact.Contact
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ContactV1TagRemove sends a request to contact-manager to remove a tag from a contact.
func (r *requestHandler) ContactV1TagRemove(ctx context.Context, contactID uuid.UUID, tagID uuid.UUID) (*cmcontact.Contact, error) {
	uri := fmt.Sprintf("/v1/contacts/%s/tags/%s", contactID, tagID)

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodDelete, "contact/contacts/<contact-id>/tags/<tag-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res cmcontact.Contact
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
```

**Step 4: Commit**

```bash
git add bin-common-handler/pkg/requesthandler/contact_*.go
git commit -m "NOJIRA-Add-contact-api-endpoints

- bin-common-handler: Add request handler methods for contact phone numbers, emails, and tags"
```

---

## Task 7: Add sendRequestContact Method and Interface Methods

**Files:**
- Modify: `bin-common-handler/pkg/requesthandler/main.go`
- Modify: `bin-common-handler/pkg/requesthandler/send_request.go` (or wherever sendRequest* methods are defined)

**Step 1: Add interface methods to main.go**

Add these method signatures to the `RequestHandler` interface:

```go
	// contact handlers
	ContactV1ContactCreate(
		ctx context.Context,
		customerID uuid.UUID,
		firstName string,
		lastName string,
		displayName string,
		company string,
		jobTitle string,
		source string,
		externalID string,
		notes string,
		phoneNumbers []cmrequest.PhoneNumberCreate,
		emails []cmrequest.EmailCreate,
		tagIDs []uuid.UUID,
	) (*cmcontact.Contact, error)
	ContactV1ContactGet(ctx context.Context, contactID uuid.UUID) (*cmcontact.Contact, error)
	ContactV1ContactList(ctx context.Context, pageToken string, pageSize uint64, filters map[cmcontact.Field]any) ([]cmcontact.Contact, error)
	ContactV1ContactUpdate(
		ctx context.Context,
		contactID uuid.UUID,
		firstName *string,
		lastName *string,
		displayName *string,
		company *string,
		jobTitle *string,
		externalID *string,
		notes *string,
	) (*cmcontact.Contact, error)
	ContactV1ContactDelete(ctx context.Context, contactID uuid.UUID) (*cmcontact.Contact, error)
	ContactV1ContactLookup(ctx context.Context, customerID uuid.UUID, phoneE164 string, email string) (*cmcontact.Contact, error)
	ContactV1PhoneNumberCreate(ctx context.Context, contactID uuid.UUID, number string, numberE164 string, phoneType string, isPrimary bool) (*cmcontact.Contact, error)
	ContactV1PhoneNumberDelete(ctx context.Context, contactID uuid.UUID, phoneNumberID uuid.UUID) (*cmcontact.Contact, error)
	ContactV1EmailCreate(ctx context.Context, contactID uuid.UUID, address string, emailType string, isPrimary bool) (*cmcontact.Contact, error)
	ContactV1EmailDelete(ctx context.Context, contactID uuid.UUID, emailID uuid.UUID) (*cmcontact.Contact, error)
	ContactV1TagAdd(ctx context.Context, contactID uuid.UUID, tagID uuid.UUID) (*cmcontact.Contact, error)
	ContactV1TagRemove(ctx context.Context, contactID uuid.UUID, tagID uuid.UUID) (*cmcontact.Contact, error)
```

Add the import:
```go
	cmcontact "monorepo/bin-contact-manager/models/contact"
	cmrequest "monorepo/bin-contact-manager/pkg/listenhandler/models/request"
```

**Step 2: Add sendRequestContact method**

Find the file where `sendRequestAgent`, `sendRequestCall`, etc. are defined (likely `send_request.go` or similar) and add:

```go
func (r *requestHandler) sendRequestContact(ctx context.Context, uri string, method sock.RequestMethod, resource string, timeout int, delay int, dataType string, data []byte) (*sock.Response, error) {
	return r.sendRequest(ctx, service.Contact, uri, method, resource, timeout, delay, dataType, data)
}
```

**Step 3: Run verification workflow for bin-common-handler**

```bash
cd bin-common-handler && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 4: Commit**

```bash
git add bin-common-handler/
git commit -m "NOJIRA-Add-contact-api-endpoints

- bin-common-handler: Add Contact interface methods and sendRequestContact"
```

---

## Task 8: Create Service Handler - Contacts

**Files:**
- Create: `bin-api-manager/pkg/servicehandler/contact.go`

**Step 1: Create contact.go**

```go
package servicehandler

import (
	"context"
	"fmt"

	amagent "monorepo/bin-agent-manager/models/agent"
	cmcontact "monorepo/bin-contact-manager/models/contact"
	cmrequest "monorepo/bin-contact-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// contactGet validates ownership and returns the contact.
func (h *serviceHandler) contactGet(ctx context.Context, contactID uuid.UUID) (*cmcontact.Contact, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "contactGet",
		"contact_id": contactID,
	})

	res, err := h.reqHandler.ContactV1ContactGet(ctx, contactID)
	if err != nil {
		log.Errorf("Could not get the contact info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// ContactCreate creates a new contact.
func (h *serviceHandler) ContactCreate(
	ctx context.Context,
	a *amagent.Agent,
	firstName string,
	lastName string,
	displayName string,
	company string,
	jobTitle string,
	source string,
	externalID string,
	notes string,
	phoneNumbers []cmrequest.PhoneNumberCreate,
	emails []cmrequest.EmailCreate,
	tagIDs []uuid.UUID,
) (*cmcontact.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ContactCreate",
		"customer_id": a.CustomerID,
		"username":    a.Username,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.ContactV1ContactCreate(ctx, a.CustomerID, firstName, lastName, displayName, company, jobTitle, source, externalID, notes, phoneNumbers, emails, tagIDs)
	if err != nil {
		log.Errorf("Could not create contact. err: %v", err)
		return nil, err
	}

	return tmp.ConvertWebhookMessage(), nil
}

// ContactGet retrieves a contact by ID.
func (h *serviceHandler) ContactGet(ctx context.Context, a *amagent.Agent, contactID uuid.UUID) (*cmcontact.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "ContactGet",
		"contact_id": contactID,
	})

	tmp, err := h.contactGet(ctx, contactID)
	if err != nil {
		log.Errorf("Could not get contact. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	return tmp.ConvertWebhookMessage(), nil
}

// ContactList retrieves a list of contacts.
func (h *serviceHandler) ContactList(ctx context.Context, a *amagent.Agent, size uint64, token string, filters map[string]string) ([]*cmcontact.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "ContactList",
		"size":  size,
		"token": token,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// Add customer_id filter
	filters["customer_id"] = a.CustomerID.String()
	filters["deleted"] = "false"

	// Convert to typed filters
	typedFilters, err := h.convertContactFilters(filters)
	if err != nil {
		log.Errorf("Could not convert filters. err: %v", err)
		return nil, err
	}

	tmps, err := h.reqHandler.ContactV1ContactList(ctx, token, size, typedFilters)
	if err != nil {
		log.Errorf("Could not list contacts. err: %v", err)
		return nil, err
	}

	res := []*cmcontact.WebhookMessage{}
	for _, tmp := range tmps {
		res = append(res, tmp.ConvertWebhookMessage())
	}

	return res, nil
}

// ContactUpdate updates a contact.
func (h *serviceHandler) ContactUpdate(
	ctx context.Context,
	a *amagent.Agent,
	contactID uuid.UUID,
	firstName *string,
	lastName *string,
	displayName *string,
	company *string,
	jobTitle *string,
	externalID *string,
	notes *string,
) (*cmcontact.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "ContactUpdate",
		"contact_id": contactID,
	})

	tmp, err := h.contactGet(ctx, contactID)
	if err != nil {
		log.Errorf("Could not get contact. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	res, err := h.reqHandler.ContactV1ContactUpdate(ctx, contactID, firstName, lastName, displayName, company, jobTitle, externalID, notes)
	if err != nil {
		log.Errorf("Could not update contact. err: %v", err)
		return nil, err
	}

	return res.ConvertWebhookMessage(), nil
}

// ContactDelete deletes a contact.
func (h *serviceHandler) ContactDelete(ctx context.Context, a *amagent.Agent, contactID uuid.UUID) (*cmcontact.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "ContactDelete",
		"contact_id": contactID,
	})

	tmp, err := h.contactGet(ctx, contactID)
	if err != nil {
		log.Errorf("Could not get contact. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	res, err := h.reqHandler.ContactV1ContactDelete(ctx, contactID)
	if err != nil {
		log.Errorf("Could not delete contact. err: %v", err)
		return nil, err
	}

	return res.ConvertWebhookMessage(), nil
}

// ContactLookup looks up a contact by phone or email.
func (h *serviceHandler) ContactLookup(ctx context.Context, a *amagent.Agent, phone string, email string) (*cmcontact.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "ContactLookup",
		"phone": phone,
		"email": email,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.ContactV1ContactLookup(ctx, a.CustomerID, phone, email)
	if err != nil {
		log.Errorf("Could not lookup contact. err: %v", err)
		return nil, err
	}

	return tmp.ConvertWebhookMessage(), nil
}

// ContactAddPhoneNumber adds a phone number to a contact.
func (h *serviceHandler) ContactAddPhoneNumber(ctx context.Context, a *amagent.Agent, contactID uuid.UUID, number string, phoneType string, isPrimary bool) (*cmcontact.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "ContactAddPhoneNumber",
		"contact_id": contactID,
	})

	tmp, err := h.contactGet(ctx, contactID)
	if err != nil {
		log.Errorf("Could not get contact. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	// Normalize phone number to E.164 format
	numberE164 := h.utilHandler.PhoneNormalize(number)

	res, err := h.reqHandler.ContactV1PhoneNumberCreate(ctx, contactID, number, numberE164, phoneType, isPrimary)
	if err != nil {
		log.Errorf("Could not add phone number. err: %v", err)
		return nil, err
	}

	return res.ConvertWebhookMessage(), nil
}

// ContactRemovePhoneNumber removes a phone number from a contact.
func (h *serviceHandler) ContactRemovePhoneNumber(ctx context.Context, a *amagent.Agent, contactID uuid.UUID, phoneNumberID uuid.UUID) (*cmcontact.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "ContactRemovePhoneNumber",
		"contact_id":      contactID,
		"phone_number_id": phoneNumberID,
	})

	tmp, err := h.contactGet(ctx, contactID)
	if err != nil {
		log.Errorf("Could not get contact. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	res, err := h.reqHandler.ContactV1PhoneNumberDelete(ctx, contactID, phoneNumberID)
	if err != nil {
		log.Errorf("Could not remove phone number. err: %v", err)
		return nil, err
	}

	return res.ConvertWebhookMessage(), nil
}

// ContactAddEmail adds an email to a contact.
func (h *serviceHandler) ContactAddEmail(ctx context.Context, a *amagent.Agent, contactID uuid.UUID, address string, emailType string, isPrimary bool) (*cmcontact.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "ContactAddEmail",
		"contact_id": contactID,
	})

	tmp, err := h.contactGet(ctx, contactID)
	if err != nil {
		log.Errorf("Could not get contact. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	res, err := h.reqHandler.ContactV1EmailCreate(ctx, contactID, address, emailType, isPrimary)
	if err != nil {
		log.Errorf("Could not add email. err: %v", err)
		return nil, err
	}

	return res.ConvertWebhookMessage(), nil
}

// ContactRemoveEmail removes an email from a contact.
func (h *serviceHandler) ContactRemoveEmail(ctx context.Context, a *amagent.Agent, contactID uuid.UUID, emailID uuid.UUID) (*cmcontact.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "ContactRemoveEmail",
		"contact_id": contactID,
		"email_id":   emailID,
	})

	tmp, err := h.contactGet(ctx, contactID)
	if err != nil {
		log.Errorf("Could not get contact. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	res, err := h.reqHandler.ContactV1EmailDelete(ctx, contactID, emailID)
	if err != nil {
		log.Errorf("Could not remove email. err: %v", err)
		return nil, err
	}

	return res.ConvertWebhookMessage(), nil
}

// ContactAddTag adds a tag to a contact.
func (h *serviceHandler) ContactAddTag(ctx context.Context, a *amagent.Agent, contactID uuid.UUID, tagID uuid.UUID) (*cmcontact.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "ContactAddTag",
		"contact_id": contactID,
		"tag_id":     tagID,
	})

	tmp, err := h.contactGet(ctx, contactID)
	if err != nil {
		log.Errorf("Could not get contact. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	res, err := h.reqHandler.ContactV1TagAdd(ctx, contactID, tagID)
	if err != nil {
		log.Errorf("Could not add tag. err: %v", err)
		return nil, err
	}

	return res.ConvertWebhookMessage(), nil
}

// ContactRemoveTag removes a tag from a contact.
func (h *serviceHandler) ContactRemoveTag(ctx context.Context, a *amagent.Agent, contactID uuid.UUID, tagID uuid.UUID) (*cmcontact.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "ContactRemoveTag",
		"contact_id": contactID,
		"tag_id":     tagID,
	})

	tmp, err := h.contactGet(ctx, contactID)
	if err != nil {
		log.Errorf("Could not get contact. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	res, err := h.reqHandler.ContactV1TagRemove(ctx, contactID, tagID)
	if err != nil {
		log.Errorf("Could not remove tag. err: %v", err)
		return nil, err
	}

	return res.ConvertWebhookMessage(), nil
}

// convertContactFilters converts map[string]string to map[cmcontact.Field]any
func (h *serviceHandler) convertContactFilters(filters map[string]string) (map[cmcontact.Field]any, error) {
	srcAny := make(map[string]any, len(filters))
	for k, v := range filters {
		srcAny[k] = v
	}

	typed, err := commondatabasehandler.ConvertMapToTypedMap(srcAny, cmcontact.Contact{})
	if err != nil {
		return nil, err
	}

	result := make(map[cmcontact.Field]any, len(typed))
	for k, v := range typed {
		result[cmcontact.Field(k)] = v
	}

	return result, nil
}
```

Add import at top:
```go
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
```

**Step 2: Commit**

```bash
git add bin-api-manager/pkg/servicehandler/contact.go
git commit -m "NOJIRA-Add-contact-api-endpoints

- bin-api-manager: Add service handler implementation for contacts"
```

---

## Task 9: Add ServiceHandler Interface Methods

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/main.go`

**Step 1: Add interface methods**

Add these to the `ServiceHandler` interface:

```go
	// contact handlers
	ContactCreate(
		ctx context.Context,
		a *amagent.Agent,
		firstName string,
		lastName string,
		displayName string,
		company string,
		jobTitle string,
		source string,
		externalID string,
		notes string,
		phoneNumbers []cmrequest.PhoneNumberCreate,
		emails []cmrequest.EmailCreate,
		tagIDs []uuid.UUID,
	) (*cmcontact.WebhookMessage, error)
	ContactGet(ctx context.Context, a *amagent.Agent, contactID uuid.UUID) (*cmcontact.WebhookMessage, error)
	ContactList(ctx context.Context, a *amagent.Agent, size uint64, token string, filters map[string]string) ([]*cmcontact.WebhookMessage, error)
	ContactUpdate(
		ctx context.Context,
		a *amagent.Agent,
		contactID uuid.UUID,
		firstName *string,
		lastName *string,
		displayName *string,
		company *string,
		jobTitle *string,
		externalID *string,
		notes *string,
	) (*cmcontact.WebhookMessage, error)
	ContactDelete(ctx context.Context, a *amagent.Agent, contactID uuid.UUID) (*cmcontact.WebhookMessage, error)
	ContactLookup(ctx context.Context, a *amagent.Agent, phone string, email string) (*cmcontact.WebhookMessage, error)
	ContactAddPhoneNumber(ctx context.Context, a *amagent.Agent, contactID uuid.UUID, number string, phoneType string, isPrimary bool) (*cmcontact.WebhookMessage, error)
	ContactRemovePhoneNumber(ctx context.Context, a *amagent.Agent, contactID uuid.UUID, phoneNumberID uuid.UUID) (*cmcontact.WebhookMessage, error)
	ContactAddEmail(ctx context.Context, a *amagent.Agent, contactID uuid.UUID, address string, emailType string, isPrimary bool) (*cmcontact.WebhookMessage, error)
	ContactRemoveEmail(ctx context.Context, a *amagent.Agent, contactID uuid.UUID, emailID uuid.UUID) (*cmcontact.WebhookMessage, error)
	ContactAddTag(ctx context.Context, a *amagent.Agent, contactID uuid.UUID, tagID uuid.UUID) (*cmcontact.WebhookMessage, error)
	ContactRemoveTag(ctx context.Context, a *amagent.Agent, contactID uuid.UUID, tagID uuid.UUID) (*cmcontact.WebhookMessage, error)
```

Add imports:
```go
	cmcontact "monorepo/bin-contact-manager/models/contact"
	cmrequest "monorepo/bin-contact-manager/pkg/listenhandler/models/request"
```

**Step 2: Generate mocks**

```bash
cd bin-api-manager && go generate ./...
```

**Step 3: Commit**

```bash
git add bin-api-manager/pkg/servicehandler/
git commit -m "NOJIRA-Add-contact-api-endpoints

- bin-api-manager: Add Contact interface methods to ServiceHandler"
```

---

## Task 10: Create Server Handlers

**Files:**
- Create: `bin-api-manager/server/contacts.go`

**Step 1: Create contacts.go**

```go
package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	cmrequest "monorepo/bin-contact-manager/pkg/listenhandler/models/request"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *server) GetContacts(c *gin.Context, params openapi_server.GetContactsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func": "GetContacts",
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)

	pageSize := uint64(100)
	if params.PageSize != nil {
		pageSize = uint64(*params.PageSize)
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 100
	}

	pageToken := ""
	if params.PageToken != nil {
		pageToken = *params.PageToken
	}

	filters := map[string]string{}

	tmps, err := h.serviceHandler.ContactList(c.Request.Context(), &a, pageSize, pageToken, filters)
	if err != nil {
		log.Errorf("Could not get contacts. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(tmps) > 0 {
		nextToken = tmps[len(tmps)-1].TMCreate
	}

	res := GenerateListResponse(tmps, nextToken)
	c.JSON(200, res)
}

func (h *server) PostContacts(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func": "PostContacts",
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)

	var req openapi_server.PostContactsJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	firstName := ""
	if req.FirstName != nil {
		firstName = *req.FirstName
	}
	lastName := ""
	if req.LastName != nil {
		lastName = *req.LastName
	}
	displayName := ""
	if req.DisplayName != nil {
		displayName = *req.DisplayName
	}
	company := ""
	if req.Company != nil {
		company = *req.Company
	}
	jobTitle := ""
	if req.JobTitle != nil {
		jobTitle = *req.JobTitle
	}
	source := ""
	if req.Source != nil {
		source = string(*req.Source)
	}
	externalID := ""
	if req.ExternalId != nil {
		externalID = *req.ExternalId
	}
	notes := ""
	if req.Notes != nil {
		notes = *req.Notes
	}

	phoneNumbers := []cmrequest.PhoneNumberCreate{}
	if req.PhoneNumbers != nil {
		for _, p := range *req.PhoneNumbers {
			pn := cmrequest.PhoneNumberCreate{}
			if p.Number != nil {
				pn.Number = *p.Number
			}
			if p.Type != nil {
				pn.Type = string(*p.Type)
			}
			if p.IsPrimary != nil {
				pn.IsPrimary = *p.IsPrimary
			}
			phoneNumbers = append(phoneNumbers, pn)
		}
	}

	emails := []cmrequest.EmailCreate{}
	if req.Emails != nil {
		for _, e := range *req.Emails {
			em := cmrequest.EmailCreate{}
			if e.Address != nil {
				em.Address = *e.Address
			}
			if e.Type != nil {
				em.Type = string(*e.Type)
			}
			if e.IsPrimary != nil {
				em.IsPrimary = *e.IsPrimary
			}
			emails = append(emails, em)
		}
	}

	tagIDs := []uuid.UUID{}
	if req.TagIds != nil {
		for _, t := range *req.TagIds {
			tagIDs = append(tagIDs, uuid.FromStringOrNil(t))
		}
	}

	res, err := h.serviceHandler.ContactCreate(c.Request.Context(), &a, firstName, lastName, displayName, company, jobTitle, source, externalID, notes, phoneNumbers, emails, tagIDs)
	if err != nil {
		log.Errorf("Could not create contact. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(201, res)
}

func (h *server) GetContactsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "GetContactsId",
		"contact_id": id,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)

	contactID := uuid.FromStringOrNil(id)
	if contactID == uuid.Nil {
		log.Error("Invalid contact ID.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.ContactGet(c.Request.Context(), &a, contactID)
	if err != nil {
		log.Errorf("Could not get contact. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutContactsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "PutContactsId",
		"contact_id": id,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)

	contactID := uuid.FromStringOrNil(id)
	if contactID == uuid.Nil {
		log.Error("Invalid contact ID.")
		c.AbortWithStatus(400)
		return
	}

	var req openapi_server.PutContactsIdJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.ContactUpdate(c.Request.Context(), &a, contactID, req.FirstName, req.LastName, req.DisplayName, req.Company, req.JobTitle, req.ExternalId, req.Notes)
	if err != nil {
		log.Errorf("Could not update contact. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteContactsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "DeleteContactsId",
		"contact_id": id,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)

	contactID := uuid.FromStringOrNil(id)
	if contactID == uuid.Nil {
		log.Error("Invalid contact ID.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.ContactDelete(c.Request.Context(), &a, contactID)
	if err != nil {
		log.Errorf("Could not delete contact. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetContactsLookup(c *gin.Context, params openapi_server.GetContactsLookupParams) {
	log := logrus.WithFields(logrus.Fields{
		"func": "GetContactsLookup",
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)

	phone := ""
	if params.Phone != nil {
		phone = *params.Phone
	}
	email := ""
	if params.Email != nil {
		email = *params.Email
	}

	if phone == "" && email == "" {
		log.Error("Either phone or email must be provided.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.ContactLookup(c.Request.Context(), &a, phone, email)
	if err != nil {
		log.Errorf("Could not lookup contact. err: %v", err)
		c.AbortWithStatus(404)
		return
	}

	c.JSON(200, res)
}

func (h *server) PostContactsIdPhoneNumbers(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "PostContactsIdPhoneNumbers",
		"contact_id": id,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)

	contactID := uuid.FromStringOrNil(id)
	if contactID == uuid.Nil {
		log.Error("Invalid contact ID.")
		c.AbortWithStatus(400)
		return
	}

	var req openapi_server.PostContactsIdPhoneNumbersJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	phoneType := ""
	if req.Type != nil {
		phoneType = string(*req.Type)
	}
	isPrimary := false
	if req.IsPrimary != nil {
		isPrimary = *req.IsPrimary
	}

	res, err := h.serviceHandler.ContactAddPhoneNumber(c.Request.Context(), &a, contactID, req.Number, phoneType, isPrimary)
	if err != nil {
		log.Errorf("Could not add phone number. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteContactsIdPhoneNumbersPhoneNumberId(c *gin.Context, id string, phoneNumberId string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteContactsIdPhoneNumbersPhoneNumberId",
		"contact_id":      id,
		"phone_number_id": phoneNumberId,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)

	contactID := uuid.FromStringOrNil(id)
	if contactID == uuid.Nil {
		log.Error("Invalid contact ID.")
		c.AbortWithStatus(400)
		return
	}

	phoneID := uuid.FromStringOrNil(phoneNumberId)
	if phoneID == uuid.Nil {
		log.Error("Invalid phone number ID.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.ContactRemovePhoneNumber(c.Request.Context(), &a, contactID, phoneID)
	if err != nil {
		log.Errorf("Could not remove phone number. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PostContactsIdEmails(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "PostContactsIdEmails",
		"contact_id": id,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)

	contactID := uuid.FromStringOrNil(id)
	if contactID == uuid.Nil {
		log.Error("Invalid contact ID.")
		c.AbortWithStatus(400)
		return
	}

	var req openapi_server.PostContactsIdEmailsJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	emailType := ""
	if req.Type != nil {
		emailType = string(*req.Type)
	}
	isPrimary := false
	if req.IsPrimary != nil {
		isPrimary = *req.IsPrimary
	}

	res, err := h.serviceHandler.ContactAddEmail(c.Request.Context(), &a, contactID, req.Address, emailType, isPrimary)
	if err != nil {
		log.Errorf("Could not add email. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteContactsIdEmailsEmailId(c *gin.Context, id string, emailId string) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "DeleteContactsIdEmailsEmailId",
		"contact_id": id,
		"email_id":   emailId,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)

	contactID := uuid.FromStringOrNil(id)
	if contactID == uuid.Nil {
		log.Error("Invalid contact ID.")
		c.AbortWithStatus(400)
		return
	}

	eID := uuid.FromStringOrNil(emailId)
	if eID == uuid.Nil {
		log.Error("Invalid email ID.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.ContactRemoveEmail(c.Request.Context(), &a, contactID, eID)
	if err != nil {
		log.Errorf("Could not remove email. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PostContactsIdTags(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "PostContactsIdTags",
		"contact_id": id,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)

	contactID := uuid.FromStringOrNil(id)
	if contactID == uuid.Nil {
		log.Error("Invalid contact ID.")
		c.AbortWithStatus(400)
		return
	}

	var req openapi_server.PostContactsIdTagsJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	tagID := uuid.FromStringOrNil(req.TagId)
	if tagID == uuid.Nil {
		log.Error("Invalid tag ID.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.ContactAddTag(c.Request.Context(), &a, contactID, tagID)
	if err != nil {
		log.Errorf("Could not add tag. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteContactsIdTagsTagId(c *gin.Context, id string, tagId string) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "DeleteContactsIdTagsTagId",
		"contact_id": id,
		"tag_id":     tagId,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)

	contactID := uuid.FromStringOrNil(id)
	if contactID == uuid.Nil {
		log.Error("Invalid contact ID.")
		c.AbortWithStatus(400)
		return
	}

	tID := uuid.FromStringOrNil(tagId)
	if tID == uuid.Nil {
		log.Error("Invalid tag ID.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.ContactRemoveTag(c.Request.Context(), &a, contactID, tID)
	if err != nil {
		log.Errorf("Could not remove tag. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
```

**Step 2: Commit**

```bash
git add bin-api-manager/server/contacts.go
git commit -m "NOJIRA-Add-contact-api-endpoints

- bin-api-manager: Add HTTP handlers for contact endpoints"
```

---

## Task 11: Run Final Verification

**Step 1: Run verification for bin-openapi-manager**

```bash
cd bin-openapi-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 2: Run verification for bin-common-handler**

```bash
cd bin-common-handler && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 3: Run verification for bin-api-manager**

```bash
cd bin-api-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 4: Verify all services that depend on bin-common-handler**

Since bin-common-handler is modified, run verification on all services:

```bash
for dir in bin-*/; do
  if [ -f "$dir/go.mod" ]; then
    echo "=== $dir ===" && \
    (cd "$dir" && \
      go mod tidy && \
      go mod vendor && \
      go generate ./... && \
      go test ./... && \
      golangci-lint run -v --timeout 5m) || echo "FAILED: $dir"
  fi
done
```

---

## Summary

This implementation adds contact API endpoints to bin-api-manager by:

1. **OpenAPI Definitions** - Added schemas and path definitions for all contact endpoints
2. **Request Handler** - Added RPC methods in bin-common-handler to communicate with contact-manager
3. **Service Handler** - Added business logic with permission checks in bin-api-manager
4. **Server Handlers** - Added HTTP handlers to expose the REST API

The implementation follows existing patterns from agents, customers, and other resources in the codebase.
