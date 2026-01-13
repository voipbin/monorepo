package file

// Field represents a typed field name for database operations
type Field string

const (
	FieldID         Field = "id"          // id
	FieldCustomerID Field = "customer_id" // customer_id
	FieldOwnerID    Field = "owner_id"    // owner_id
	FieldAccountID  Field = "account_id"  // account_id

	FieldReferenceType Field = "reference_type" // reference_type
	FieldReferenceID   Field = "reference_id"   // reference_id

	FieldName   Field = "name"   // name
	FieldDetail Field = "detail" // detail

	FieldBucketName Field = "bucket_name" // bucket_name
	FieldFilename   Field = "filename"    // filename
	FieldFilepath   Field = "filepath"    // filepath
	FieldFilesize   Field = "filesize"    // filesize

	FieldURIBucket   Field = "uri_bucket"   // uri_bucket
	FieldURIDownload Field = "uri_download" // uri_download

	FieldTMDownloadExpire Field = "tm_download_expire" // tm_download_expire
	FieldTMCreate         Field = "tm_create"          // tm_create
	FieldTMUpdate         Field = "tm_update"          // tm_update
	FieldTMDelete         Field = "tm_delete"          // tm_delete

	// filter only
	FieldDeleted Field = "deleted"
)
