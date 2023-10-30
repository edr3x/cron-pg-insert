package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

func init() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("Error loading .env file")
	}
}

func main() {
	engine := html.New("./views", ".html")
	app := fiber.New(fiber.Config{
		Views: engine,
	})
	app.Get("/", func(c *fiber.Ctx) error {
		return c.Render("index", fiber.Map{})
	})

	app.Get("/ws", websocket.New(func(c *websocket.Conn) {
		var (
			mt  = websocket.TextMessage
			msg []byte
			err error
		)

		dataChannel := make(chan []byte)

		go func() {
			counter := 1
			for {
				timestamp := time.Now().Unix()

				userName := fmt.Sprintf("User_%d", timestamp)
				userEmail := fmt.Sprintf("user%d@example.com", counter)

				conn, err := pgx.Connect(context.Background(), os.Getenv("INSERT_DATABASE_URL"))
				if err != nil {
					log.Println("Database connection failed:", err)
					dataChannel <- []byte(strings.Join([]string{"Insert Database connection failed:", err.Error()}, " "))
					return
				}
				defer conn.Close(context.Background())

				_, err = conn.Exec(context.Background(), "INSERT INTO \"user\" (\"name\", \"email\") VALUES ($1, $2);", userName, userEmail)
				if err != nil {
					log.Println("Insert failed:", err)
					dataChannel <- []byte(strings.Join([]string{"Insert failed:", err.Error()}, " "))
					return
				}
				counter++
				time.Sleep(3 * time.Second)
			}
		}()

		go func() {
			for {
				var userNames []string

				dbConn, err := pgx.Connect(context.Background(), os.Getenv("READ_DATABASE_URL"))
				if err != nil {
					fmt.Fprintf(os.Stderr, "Read database Unable to connect %v\n", err)
					os.Exit(1)
				}
				defer dbConn.Close(context.Background())

				rows, err := dbConn.Query(context.Background(), "SELECT \"name\", \"created_at\" FROM \"user\";")
				if err != nil {
					log.Println("Query failed:", err)
					dataChannel <- []byte(strings.Join([]string{"Query failed:", err.Error()}, " "))
					return
				}
				defer rows.Close()

				for rows.Next() {
					var userName string
					var created_at time.Time
					if err := rows.Scan(&userName, &created_at); err != nil {
						log.Println("Scan failed:", err)
						dataChannel <- []byte(strings.Join([]string{"Scan failed:", err.Error()}, " "))
						return
					}
					userNames = append(userNames, userName+" -- "+created_at.String()+"\n")
				}

				if err := rows.Err(); err != nil {
					log.Println("Row iteration failed:", err)
					dataChannel <- []byte(strings.Join([]string{"Row iteration failed:", err.Error()}, " "))
					return
				}

				dataChannel <- []byte(strings.Join(userNames, ", "))

				time.Sleep(3 * time.Second)
			}
		}()

		for {
			msg = <-dataChannel

			if err = c.WriteMessage(mt, msg); err != nil {
				log.Println("write:", err)
				break
			}
		}
	}))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Fatal(app.Listen("0.0.0.0:" + port))
}
