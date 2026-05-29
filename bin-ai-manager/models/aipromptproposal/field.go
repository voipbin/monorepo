package aipromptproposal

// Field represents AIPromptProposal column for database queries.
type Field string

// List of fields.
const (
	FieldID         Field = "id"
	FieldCustomerID Field = "customer_id"

	FieldAIID                   Field = "ai_id"
	FieldBasisPromptHistoryID   Field = "basis_prompt_history_id"
	FieldAppliedPromptHistoryID Field = "applied_prompt_history_id"

	FieldStatus Field = "status"
	FieldError  Field = "error"

	FieldTMCreate Field = "tm_create"
	FieldTMUpdate Field = "tm_update"
	FieldTMDelete Field = "tm_delete"

	FieldDeleted Field = "deleted" // synthetic; translated by ApplyFields to tm_delete IS NULL / IS NOT NULL
)
