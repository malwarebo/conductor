package ctxkeys

type Key string

const (
	UserID         Key = "user_id"
	UserEmail      Key = "user_email"
	UserRoles      Key = "user_roles"
	APIKey         Key = "api_key"
	EncryptedData  Key = "encrypted_data"
	TenantID       Key = "tenant_id"
	Tenant         Key = "tenant"
	IdempotencyKey Key = "idempotency_key"
)
