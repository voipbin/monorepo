package billingaccounts

import "github.com/gin-gonic/gin"

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	billingAccounts := r.Group("/billing_accounts")

	billingAccounts.GET("", billingaccountsGET)
	billingAccounts.POST("", billingAccountsPOST)

	billingAccounts.DELETE("/:id", billingAccountsIDDelete)
	billingAccounts.GET("/:id", billingAccountsIDGET)
	billingAccounts.PUT("/:id", billingAccountsIDPut)

	billingAccounts.PUT("/:id/payment_info", billingAccountsIDPaymentInfoPut)
	billingAccounts.POST("/:id/balance_add_force", billingAccountsIDBalanceAddForcePOST)
	billingAccounts.POST("/:id/balance_subtract_force", billingAccountsIDBalanceSubtractForcePOST)
}
