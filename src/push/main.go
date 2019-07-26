package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"intercom"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// HTTP Rest API for pushing
func main() {
	//pushURL := "http://internal-localalbElasticlogQuery-uw2-620013642.us-west-2.elb.amazonaws.com/push"
	pushURL := "http://127.0.0.1/push"
	contentType := "application/json"

	users := make([]string, 1)
	for i := range users {
		users[i] = strconv.Itoa(i)
	}

	for {
		for i := range users {
			pm := intercom.CommMessage{
				UserID:  users[i],
				CommID:  uuid.New().String(),
				Message: fmt.Sprintf("Hello user[%s], it is now: %s", users[i], time.Now().Format("2006-01-02 15:04:05.000")),
			}
			b, _ := json.Marshal(pm)

			resp, err := http.DefaultClient.Post(pushURL, contentType, bytes.NewReader(b))
			if err == nil {

				body, ok := ioutil.ReadAll(resp.Body)
				if ok == nil {
					fmt.Println(string(body))
					resp.Body.Close()
					//time.Sleep(time.Second)
				} else {
					fmt.Println(ok.Error())
				}
			} else {
				fmt.Println(err.Error())
			}
		}
		time.Sleep(time.Second * 10)
	}
}
