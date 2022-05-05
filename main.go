package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func main(){
	router := gin.Default())
	router.POST("routes/payments/")
	router.GET("routes/payments/:id")
	router.GET("routes/payments/") // Fetch all payments
	router.Run("localhost:8080")
}

