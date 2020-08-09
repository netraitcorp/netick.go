package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/url"

	"github.com/gorilla/websocket"
	"github.com/netraitcorp/netick.go/pb"
	"google.golang.org/protobuf/proto"
)

var addr = flag.String("addr", "127.0.0.1:2634", "netick websocket server address")

func init() {
	flag.Parse()
}

func main() {
	u := url.URL{Scheme: "ws", Host: *addr}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Printf("[ERROR] ws connect failed, %s", err.Error())
		return
	}

	ctx, cancelCtx := context.WithCancel(context.Background())
	readCtx, _ := context.WithCancel(ctx)
	writeCtx, _ := context.WithCancel(ctx)

	errChan := make(chan error, 2)
	wb := make(chan []byte, 0xFF)

	go func(ctx context.Context) {
		for {
			msgType, message, err := c.ReadMessage()
			select {
			case <-ctx.Done():
				return
			default:
			}

			if err != nil {
				errChan <- err
				return
			}
			if msgType != websocket.BinaryMessage {
				errChan <- errors.New("recv data not BinaryMessage")
				return
			}
			log.Printf("[INFO] recv data: %s", message)
		}
	}(readCtx)

	go func(ctx context.Context) {
		for {
			select {
			case data := <-wb:
				if err := c.WriteMessage(websocket.BinaryMessage, data); err != nil {
					errChan <- err
					return
				}
				log.Printf("[INFO] send data: %s", data)
			case <-ctx.Done():
				return
			}
		}
	}(writeCtx)

	pack := &pb.AuthReq{
		Password: "7c4a8d09ca3762af61e59520943dc26494f8941b",
	}
	data, err := proto.Marshal(pack)
	if err != nil {
		log.Fatalf("[ERROR] proto.Marshal: %s", err.Error())
		return
	}
	data = append([]byte{0x04}, data...)
	wb <- data

	if err := <-errChan; err != nil {
		log.Printf("[ERROR] rw error, %s", err.Error())
	}
	close(errChan)

	cancelCtx()
}
