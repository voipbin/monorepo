# bin-tag-manager Domain

## Domain Entities

### Tag
A customer-scoped label with a `name` and optional `detail`. Tags provide a cross-resource categorization mechanism used by:
- `bin-contact-manager` — tag contacts for segmentation
- `bin-queue-manager` — tag queues for routing logic
- `bin-campaign-manager` — tag campaigns

Fields:
- `id` — UUID, primary key
- `customer_id` — owning customer UUID
- `name` — tag name (unique per customer)
- `detail` — optional description
- `tm_create` — creation timestamp
- `tm_update` — last update timestamp
- `tm_delete` — soft-delete sentinel (`9999-01-01` = active)

## Key Business Rules

- **Customer isolation.** Tags are strictly scoped by `customer_id`. A tag from one customer is never visible to another.
- **Unique name per customer.** Two tags for the same customer cannot share the same `name`. The DB enforces this at the unique index level.
- **Soft deletes.** Tags are never physically removed by the DELETE endpoint; `tm_delete` is set to the current time. Only the cascading customer delete may trigger a permanent removal depending on implementation.
- **Cascading deletes on customer removal.** When `bin-customer-manager` publishes `customer_deleted`, all tags belonging to that customer are deleted and `tag_deleted` events are emitted for each.
- **Event-driven consumers.** Services that embed tag data in their own records should subscribe to `bin-manager.tag-manager.event` and update or invalidate their local copies when `tag_updated` or `tag_deleted` is received.
- **Cache invalidation on every mutation.** Create, update, and delete operations invalidate the Redis keys for both the individual tag and the customer's tag list.
