package aiaudit

// Field represents AIAudit column for database queries
type Field string

// List of fields
const (
	FieldID         Field = "id"
	FieldCustomerID Field = "customer_id"

	FieldAIcallID        Field = "aicall_id"
	FieldAIID            Field = "ai_id"
	FieldPromptHistoryID Field = "prompt_history_id"

	FieldStatus       Field = "status"
	FieldOverallScore Field = "overall_score"
	FieldEvaluation   Field = "evaluation"
	FieldLanguage     Field = "language"
	FieldError        Field = "error"

	FieldTMCreate Field = "tm_create"
	FieldTMUpdate Field = "tm_update"
	FieldTMDelete Field = "tm_delete"

	FieldDeleted Field = "deleted"
)
