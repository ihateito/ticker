package main

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/websocket"
	"strings"
	"time"
)

type Message struct {
	Type       string           `json:"type"`
	ProductID  string           `json:"product_id"`
	ProductIds []string         `json:"product_ids"`
	BestBid    string           `json:"best_bid"`
	BestAsk    string           `json:"best_ask"`
	Channels   []MessageChannel `json:"channels"`
}

type MessageChannel struct {
	Name       string   `json:"name"`
	ProductID  string   `json:"product_id"`
	ProductIds []string `json:"product_ids"`
}

var TickerChan = make(chan string)

var Db = getDb()

func main() {
	go startTicker([]string{"ETH-BTC"})
	go startTicker([]string{"BTC-USD"})
	go startTicker([]string{"BTC-EUR"})

	for {
		message, opened := <-TickerChan

		if !opened {
			break
		}

		fmt.Println(message)
	}
}

func startSubscribe(wsConn *websocket.Conn, message *Message) (*Message, error) {
	var receivedMessage Message

	if err := wsConn.WriteJSON(message); err != nil {
		return nil, err
	}

	for {
		if err := wsConn.ReadJSON(&receivedMessage); err != nil {
			return nil, err
		}

		if receivedMessage.Type != "subscriptions" {
			break
		}
	}

	return &receivedMessage, nil
}

func startTicker(productIds []string) {
	wsConn, err := getWebsocketClient()

	if err != nil {
		panic(err)
	}

	subscribe := Message{
		Type:       "subscribe",
		ProductIds: productIds,
		Channels: []MessageChannel{
			MessageChannel{
				Name:       "ticker",
				ProductIds: productIds,
			},
		},
	}

	message, err := startSubscribe(wsConn, &subscribe)

	if err != nil {
		panic(err)
	}

	if message.Type != "ticker" {
		panic(errors.New("Invalid message type: " + message.Type))
	}

	for {
		message := Message{}

		if err := wsConn.ReadJSON(&message); err != nil {
			panic(err)
		}

		TickerChan <- message.Type + " " + message.ProductID + " " + message.BestBid + " " + message.BestAsk

		_, err = Db.Exec(
			"INSERT INTO ticks (`timestamp`, symbol, bid, ask) VALUES (?, ?, ?, ?)",
			time.Now().UnixNano()/(int64(time.Millisecond)/int64(time.Nanosecond)),
			strings.Replace(message.ProductID, "-", "", 1),
			message.BestBid,
			message.BestAsk)

		if err != nil {
			panic(err)
		}
	}
}

func getWebsocketClient() (*websocket.Conn, error) {
	var wsDialer websocket.Dialer
	wsConn, _, err := wsDialer.Dial("wss://ws-feed-public.sandbox.pro.coinbase.com", nil)

	return wsConn, err
}

func getDb() *sql.DB {
	db, err := sql.Open("mysql", "docker:docker@tcp(db:3306)/ticker")

	if err != nil {
		panic(err)
	}

	return db
}
