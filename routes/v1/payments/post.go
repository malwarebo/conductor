// route entry point
func createCharge(c *gin.Context) {
	var newCharge charge

	if err := c.BindJSON(&newCharge); err != nil {
		return
	}

	charges = append(charges, newCharge)
	c.IndentedJSON(http.StatusCreated, newCharge)
}