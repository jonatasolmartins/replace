package main

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/template/django/v3"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

func main() {

	app := initServer()

	app.Use(recover.New())

	app.Get("/api/index", handleDefault)

	app.Post("/api/create", handleCreate)

	log.Fatal(app.Listen(":3000"))
}

func GenerateKeys() (string, string) {
	sk := nostr.GeneratePrivateKey()
	pk, _ := nostr.GetPublicKey(sk)
	nsec, _ := nip19.EncodePrivateKey(sk)
	npub, _ := nip19.EncodePublicKey(pk)

	fmt.Println(nsec)
	fmt.Println(npub)

	return nsec, npub
}

func initServer() *fiber.App {

	engine := initEngine()

	app := fiber.New(fiber.Config{
		Views: engine,
		ErrorHandler: func(ctx *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			//message := err.Error()

			var e *fiber.Error

			if errors.As(err, &e) {
				code = e.Code
			}

			err = ctx.Status(code).SendFile(fmt.Sprintf("./views/errors/%d.html", code))

			if err != nil {
				return ctx.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
			}

			return nil
		},
	})

	return app
}

func initEngine() *django.Engine {
	engine := django.New("./views", ".html")

	engine.AddFunc("parseLineBreak", func(value interface{}) string {
		if str, ok := value.(string); ok {
			return strings.Replace(str, "\n", "<br />", -1)
		}
		return ""
	})

	fm := map[string]interface{}{
		"lower": strings.ToLower,
		"upper": strings.ToUpper,
	}

	engine.AddFuncMap(fm)

	if err := engine.Load(); err != nil {
		log.Fatal("load:", err)
	}

	return engine
}

func handleDefault(c *fiber.Ctx) error {
	return c.Render("index", nil)
}

func handleCreate(c *fiber.Ctx) error {
	context := fiber.Map{
		"greetings": "",
		"npub":      "",
		"nsec":      "",
	}

	nsec, npub := GenerateKeys()

	context["nsec"] = nsec
	context["npub"] = npub

	context["greetings"] = fmt.Sprintf("Hello %s", c.FormValue("fname"))

	return c.Render("index", context)
}
