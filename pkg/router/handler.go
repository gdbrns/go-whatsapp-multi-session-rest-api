package router

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
)

func HttpErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := err.Error()
	if fiberErr, ok := err.(*fiber.Error); ok {
		code = fiberErr.Code
		message = fiberErr.Message
	}
	response := &ResError{
		Status: false,
		Code:   code,
		Error:  fmt.Sprintf("%v", message),
	}
	logError(c, response.Code, response.Error)
	return c.Status(response.Code).JSON(response)
}
