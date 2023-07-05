package main

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/idempotency"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/template/django/v3"
	"github.com/joho/godotenv"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"github.com/sujit-baniya/flash"
)

func main() {
	app, err := initApp()
	if err != nil {
		log.Fatal(err)
	}

	app.Use(helmet.New())
	app.Use(idempotency.New())
	app.Use(recover.New())

	//app.Use(cors.New())
	app.Use(logger.New(logger.Config{
		Format: "[${ip}]:${port} ${status} - ${method} ${path}\n",
	}))

	app.Static("src/static", "./src/static")
	// app.Use(favicon.New(favicon.Config{
	// 	File: "./favicon.ico",
	// 	URL:  "/favicon.ico",
	// }))

	app.Use(WithFlash)

	app.Get("/index", handleDefault)

	account := app.Group("/account")
	account.Post("/create", handleCreate)

	feed := app.Group("/feed")
	feed.Get("/", handleFeed)
	feed.Get("won/:npub", handleFeed)
	feed.Post("/post", func(c *fiber.Ctx) error { return c.Redirect("<h1>To be implemented</h1>") })

	log.Fatal(app.Listen(os.Getenv("HTTP_LISTEN_ADDR")))
}

func GenerateKeys() (string, string) {
	sk := nostr.GeneratePrivateKey()
	pk, _ := nostr.GetPublicKey(sk)
	nsec, _ := nip19.EncodePrivateKey(sk)
	npub, _ := nip19.EncodePublicKey(pk)

	return nsec, npub
}

func initApp() (*fiber.App, error) {
	if err := godotenv.Load(); err != nil {
		return nil, err
	}

	engine := initEngineTemplate()

	app := fiber.New(fiber.Config{
		ErrorHandler:          ErrorHandler,
		DisableStartupMessage: false,
		PassLocalsToViews:     true,
		Views:                 engine,
	})

	return app, nil
}

func ErrorHandler(ctx *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError

	var e *fiber.Error
	message := err.Error()
	//TODO: remove this line
	println(message)

	if errors.As(err, &e) {
		code = e.Code
	}

	err = ctx.Status(code).SendFile(fmt.Sprintf("./views/errors/%d.html", code))

	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
	}

	return nil
}

func initEngineTemplate() *django.Engine {
	engine := django.New("./src/views", ".html")
	engine.Reload(true)

	engine.AddFunc("formatTime", func(t time.Time) string {
		timeZero := time.Time{}
		if t.Equal(timeZero) {
			return "n/a"
		}
		return t.Format(time.DateTime)
	})

	fm := map[string]interface{}{
		"lower": strings.ToLower,
		"upper": strings.ToUpper,
	}

	engine.AddFunc("parseLineBreak", func(value interface{}) string {
		if str, ok := value.(string); ok {
			return strings.Replace(str, "\n", "<br />", -1)
		}
		return ""
	})

	engine.AddFunc("css", func(name string) (res template.HTML) {
		filepath.Walk("src/static/styles", func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.Name() == name {
				res = template.HTML("<link rel=\"stylesheet\" href=\"/" + path + "\">")
			}
			return nil
		})
		return
	})

	engine.AddFuncMap(fm)

	return engine
}

func handleDefault(c *fiber.Ctx) error {
	return c.Render("index", flash.Get(c))
}

func handleCreate(c *fiber.Ctx) error {
	context := fiber.Map{
		"npub": "",
		"nsec": "",
	}
	//TODO: remove this line
	fmt.Println(c.FormValue("lkey"))
	nsec, npub := GenerateKeys()

	context["nsec"] = nsec
	context["npub"] = npub

	return flash.WithData(c, context).Redirect("/index")
}

func handleFeed(c *fiber.Ctx) error {
	ctx := context.Background()
	relay, err := nostr.RelayConnect(ctx, "wss://relay.damus.io/")
	if err != nil {
		panic(err)
	}

	npub := "npub1ezyhugdszsnq4yjwxuy3ml8qnjmls3y3dz4p9h04tzhjyqmtv0xqxvmwhq"

	var filters nostr.Filters
	if _, v, err := nip19.Decode(npub); err == nil {
		pub := v.(string)
		filters = []nostr.Filter{{
			Kinds:   []int{nostr.KindTextNote},
			Authors: []string{pub},
			Limit:   1,
		}}
	} else {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	sub, err := relay.Subscribe(ctx, filters)
	if err != nil {
		panic(err)
	}

	var content string
	for ev := range sub.Events {
		fmt.Println(ev.ID)
		fmt.Println(ev.Content)
		content = ev.Content
		break
	}

	return c.Render("feed", fiber.Map{"content": content})
}

func WithFlash(c *fiber.Ctx) error {
	values := flash.Get(c)
	c.Locals("flash", values)
	return c.Next()
}
