func fetchPayment(c *gin.Context){
	id := c.Param("id")

	for _, a := range charges {
		if a.ID == id {
			c.IndentedJSON(http.StatusOK, a)
			return
		}
	}
	c.IndentedJSON(http.StatusNotFound, gin.H{"error": "Payment not found"})
}


func fetchAllPayments(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, charges)
}
