package api

import "github.com/gin-gonic/gin"

func GetPatient(c *gin.Context) {
    c.JSON(200, gin.H{"message": "Patient endpoint"})
}
