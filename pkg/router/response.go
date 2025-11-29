package router

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/log"
)

type ResSuccess struct {
	Status  bool   `json:"status"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type ResSuccessWithData struct {
	Status  bool        `json:"status"`
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type ResError struct {
	Status bool   `json:"status"`
	Code   int    `json:"code"`
	Error  string `json:"error"`
}

func logSuccess(c *fiber.Ctx, code int, message string) {
	statusMessage := http.StatusText(code)

	if statusMessage == message || c.OriginalURL() == BaseURL {
		log.Print(c).Info(fmt.Sprintf("%d %v", code, statusMessage))
	} else {
		log.Print(c).Info(fmt.Sprintf("%d %v", code, message))
	}
}

func logError(c *fiber.Ctx, code int, message string) {
	statusMessage := http.StatusText(code)

	if statusMessage == message {
		log.Print(c).Error(fmt.Sprintf("%d %v", code, statusMessage))
	} else {
		log.Print(c).Error(fmt.Sprintf("%d %v", code, message))
	}
}

func ResponseSuccess(c *fiber.Ctx, message string) error {
	var response ResSuccess

	response.Status = true
	response.Code = http.StatusOK

	if strings.TrimSpace(message) == "" {
		message = http.StatusText(response.Code)
	}
	response.Message = message

	logSuccess(c, response.Code, response.Message)
	return c.Status(response.Code).JSON(response)
}

func ResponseSuccessWithData(c *fiber.Ctx, message string, data interface{}) error {
	var response ResSuccessWithData

	response.Status = true
	response.Code = http.StatusOK

	if strings.TrimSpace(message) == "" {
		message = http.StatusText(response.Code)
	}
	response.Message = message
	response.Data = data

	logSuccess(c, response.Code, response.Message)
	return c.Status(response.Code).JSON(response)
}

	
func ResponseSuccessWithHTML(c *fiber.Ctx, html string) error {
	logSuccess(c, http.StatusOK, http.StatusText(http.StatusOK))
	c.Type("html", "utf-8")
	return c.Status(http.StatusOK).SendString(html)
}

func ResponseCreated(c *fiber.Ctx, message string) error {
	var response ResSuccess

	response.Status = true
	response.Code = http.StatusCreated

	if strings.TrimSpace(message) == "" {
		message = http.StatusText(response.Code)
	}
	response.Message = message

	logSuccess(c, response.Code, response.Message)
	return c.Status(response.Code).JSON(response)
}

func ResponseNoContent(c *fiber.Ctx) error {
	return c.SendStatus(http.StatusNoContent)
}

func ResponseNotFound(c *fiber.Ctx, message string) error {
	var response ResError

	response.Status = false
	response.Code = http.StatusNotFound

	if strings.TrimSpace(message) == "" {
		message = http.StatusText(response.Code)
	}
	response.Error = message

	logError(c, response.Code, response.Error)
	return c.Status(response.Code).JSON(response)
}

func ResponseAuthenticate(c *fiber.Ctx) error {
	c.Set("WWW-Authenticate", `Basic realm="Authentication Required"`)
	return ResponseUnauthorized(c, "")
}

func ResponseUnauthorized(c *fiber.Ctx, message string) error {
	var response ResError

	response.Status = false
	response.Code = http.StatusUnauthorized

	if strings.TrimSpace(message) == "" {
		message = http.StatusText(response.Code)
	}
	response.Error = message

	logError(c, response.Code, response.Error)
	return c.Status(response.Code).JSON(response)
}

func ResponseBadRequest(c *fiber.Ctx, message string) error {
	var response ResError

	response.Status = false
	response.Code = http.StatusBadRequest

	if strings.TrimSpace(message) == "" {
		message = http.StatusText(response.Code)
	}
	response.Error = message

	logError(c, response.Code, response.Error)
	return c.Status(response.Code).JSON(response)
}

func ResponseInternalError(c *fiber.Ctx, message string) error {
	var response ResError

	response.Status = false
	response.Code = http.StatusInternalServerError

	if strings.TrimSpace(message) == "" {
		message = http.StatusText(response.Code)
	}
	response.Error = message

	logError(c, response.Code, response.Error)
	return c.Status(response.Code).JSON(response)
}

func ResponseBadGateway(c *fiber.Ctx, message string) error {
	var response ResError

	response.Status = false
	response.Code = http.StatusBadGateway

	if strings.TrimSpace(message) == "" {
		message = http.StatusText(response.Code)
	}
	response.Error = message

	logError(c, response.Code, response.Error)
	return c.Status(response.Code).JSON(response)
}
