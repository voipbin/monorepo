# monitoring

This directory contains the API validator test suite and Grafana dashboard provisioning files.

## Grafana Dashboards

**All Grafana dashboard JSON provisioning files MUST be placed in `monitoring/grafana/dashboards/`.**

- File naming: `<service-name>.json` (e.g., `flow-manager.json`, `call-manager.json`)
- One dashboard per service
- Dashboards are importable JSON files (Grafana provisioning format)

## API Testing Guidelines (monitoring/api-validator)

### Excluded Test Cases - Cost-Sensitive Operations

**NEVER create tests for the following API endpoints as they incur real costs:**

- ❌ **Number Buy** (`POST /numbers`) - Costs money to purchase/order phone numbers
- ❌ **Call Create to external numbers** (`POST /calls` to PSTN numbers) - Costs money to make actual phone calls
- ✅ **Call Create to virtual numbers** (`POST /calls` to virtual/internal numbers) - Allowed for testing (e.g., generating recordings for STT tests)
- ❌ **Email Send** (`/emails/send` or similar) - Costs money to send emails
- ❌ **Message Send** (`/messages/send` or similar) - Costs money to send SMS/messages

### Allowed Test Operations

✅ **Read-only operations** are safe and encouraged:
- GET requests (list, retrieve, search)
- Pagination and filtering tests
- Schema validation tests

✅ **CRUD operations on test resources** are allowed if they don't trigger external actions:
- Creating/updating/deleting test agents, campaigns, queues, etc.
- As long as they don't trigger actual calls, emails, or purchases

### Before Adding New Tests

When adding tests for a new API resource, always verify:
1. Does this operation cost money? (calls, SMS, emails, number purchases)
2. Does this trigger external actions that can't be undone?
3. If unsure, ask the user before implementing
