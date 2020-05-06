package main

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	ws "github.com/gorilla/websocket"
	"reflect"
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

func startSubscribe(wsConn *ws.Conn, message *Message) (*Message, error) {
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

func Ensure(a interface{}) error {
	field := reflect.Indirect(reflect.ValueOf(a))

	switch field.Kind() {
	case reflect.Slice:
		if reflect.ValueOf(field.Interface()).Len() == 0 {
			return fmt.Errorf(fmt.Sprintf("Slice is zero"))
		}
	default:
		if reflect.Zero(field.Type()).Interface() == field.Interface() {
			return fmt.Errorf(fmt.Sprintf("Property is zero"))
		}
	}

	return nil
}

func EnsureProperties(a interface{}, properties []string) error {
	valueOf := reflect.ValueOf(a)

	for _, property := range properties {
		field := reflect.Indirect(valueOf).FieldByName(property)

		if err := Ensure(field.Interface()); err != nil {
			return fmt.Errorf(fmt.Sprintf("%s: %s", err.Error(), property))
		}
	}

	return nil
}

func NewTestWebsocketClient() (*ws.Conn, error) {
	var wsDialer ws.Dialer
	wsConn, _, err := wsDialer.Dial("wss://ws-feed-public.sandbox.pro.coinbase.com", nil)

	return wsConn, err
}

func startTicker(productIds []string) {
	wsConn, err := NewTestWebsocketClient()

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

	props := []string{"Type", "ProductID", "BestBid", "BestAsk"}
	if err := EnsureProperties(message, props); err != nil {
		panic(err)
	}

	db, err := sql.Open("mysql", "root:root@tcp(local_db:3306)/my_fx")

	if err != nil {
		panic(err)
	}

	for true {
		message := Message{}
		if err := wsConn.ReadJSON(&message); err != nil {
			println(err.Error())
			break
		}

		TickerChan <- message.Type + " " + message.ProductID + " " + message.BestBid + " " + message.BestAsk

		_, err := db.Exec(
			"INSERT INTO ticks (`timestamp`, symbol, bid, ask) VALUES (?, ?, ?, ?)",
			time.Now().Unix(),
			strings.Replace(message.ProductID, "-", "", 1),
			message.BestBid,
			message.BestAsk)

		if err != nil {
			panic(err)
		}
	}
}

var TickerChan = make(chan string)

func main() {
	go startTicker([]string{"ETH-BTC"})
	go startTicker([]string{"BTC-USD"})
	go startTicker([]string{"BTC-EUR"})

	for true {
		message, opened := <-TickerChan

		if !opened {
			break
		}

		fmt.Println(message)
	}
}
