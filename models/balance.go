package models

type Balance struct {
	Available    int64  `json:"available"`
	Pending      int64  `json:"pending"`
	Currency     string `json:"currency"`
	ProviderName string `json:"provider_name"`
}

type BalanceResponse struct {
	Balances []*Balance `json:"balances"`
}

