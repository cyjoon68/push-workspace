package domain

const (
	PlanFree  = "FREE"
	PlanPro   = "PRO"
	PlanUltra = "ULTRA"
)

const (
	SubStatusNone     = "NONE"
	SubStatusActive   = "ACTIVE"
	SubStatusCanceled = "CANCELED"
)

const (
	LedgerPurchase = "PURCHASE"
	LedgerRefund   = "REFUND"
)

var PlanCreditsMicroCredits = map[string]int64{
	PlanPro:   2_000_000,
	PlanUltra: 10_000_000,
}

func ValidPlan(p string) bool {
	return p == PlanFree || p == PlanPro || p == PlanUltra
}
