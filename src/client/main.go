package main

import (
	"encoding/json"
	"flag"
	"github.com/gorilla/websocket"
	"intercom"
	"log"
	"net/http"
	"strconv"
	"sync"
)

func main() {

	var url = flag.String("url", "ws://127.0.0.1:80/ws", "websocket server: ws://address:port/path")
	var concurrency = flag.Int("number", 10, "the number of concurrent clients")

	flag.Parse()

	done := make(chan struct{})

	// get ^C from the terminal
	/*go func(){
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c
		log.Println("exiting the program...")
		os.Exit(1)
	}()
	*/
	var wg sync.WaitGroup
	wg.Add(*concurrency)

	users := make([]string, *concurrency)

	locks := make(chan struct{}, 10)

	for i := range users {
		users[i] = strconv.Itoa(i)
		//start the client
		//log.Println("start to new connection: *s", users[i])
		go func(user string) {
			defer wg.Done()
			locks <- struct{}{}
			newEchoClient(*url, user, done, locks)
		}(users[i])
	}
	// wait for termination
	log.Println("finished new clients")
	wg.Wait()
}

func newEchoClient(url, user string, done, limit chan struct{}) error {
	hd := http.Header{}
	hd.Add("CE-X-USER", user)

	c, _, err := websocket.DefaultDialer.Dial(url, hd)
	if err != nil {
		log.Fatal("dial:", err, user)
		return err
	}

	defer c.Close()
	/*
		rm := intercom.RegisterMessage{
			Token: user,
			Event: "what ever",
		}
		strRm, _ := json.Marshal(rm)
		msg := intercom.WSMessage{
			Kind: intercom.RegisterMessageType,
			Body: string(strRm),
		}

		strMsg, _ := json.Marshal(msg)

		err = c.WriteMessage(websocket.TextMessage, []byte(strMsg))

		if err != nil {
			log.Println("write:", err)
			c.Close()
			return err
		} */

	log.Println("connected", user)
	<-limit

	// wait until the server close the connection
readloop:
	for {
		select {
		case <-done:
			break readloop
		default:

			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return err
			}
			log.Printf("recv: %s", message)
			wsm := intercom.WSMessage{
				Kind: intercom.NormalMessageType,
				Body: string(message),
			}

			strMsg, _ := json.Marshal(wsm)
			err = c.WriteMessage(websocket.TextMessage, []byte(strMsg))
			if err != nil {
				return err
			}

		}
	}

	return nil

}
