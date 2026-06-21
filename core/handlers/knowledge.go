package handlers

// Shared helpers for the migrated knowledge-base CRUD handlers
// (courses, mechanisms, crm, sessions).

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// parseUintParam reads a uint path parameter (e.g. /:id).
func parseUintParam(c *gin.Context, name string) (uint, error) {
	v, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(v), nil
}
