package api

import "github.com/gin-gonic/gin"

func GetDoctors(c *gin.Context) {
    c.JSON(200, gin.H{"message": "Doctors endpoint"})
}
