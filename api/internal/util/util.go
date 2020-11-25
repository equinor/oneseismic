package util

import (
	"github.com/google/uuid"
	"github.com/gin-gonic/gin"
)

func MakePID() string {
	return uuid.New().String()
}

func GeneratePID(ctx *gin.Context) {
	ctx.Set("pid", MakePID())
}
