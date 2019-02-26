package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/go-ini/ini"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

type PushOverMessage struct {
	Token   string `json:"token"`
	User    string `json:"user"`
	Message string `json:"message"`
	Title   string `json:"title"`
}

type PushOverResponse struct {
	Status  int    `json:"status"`
	Request string `json:"request"`
}

func firstExists(files ...string) string {
	for _, file := range files {
		if _, err := os.Stat(file); err == nil {
			return file
		}
	}

	return ""
}

func pushOverMessage(cfg *ini.File, confPath string) ([]byte, error) {
	var result []byte

	pushOver, err := cfg.GetSection("pushover")

	if err != nil {
		return result, fmt.Errorf("failed to read 'pushover' section of %s: %v", confPath, err)
	}

	userKey, err := pushOver.GetKey("user_key")

	if err != nil {
		return result, fmt.Errorf("faild to get user_key: %v", err)
	}

	appKey, err := pushOver.GetKey("app_key")

	if err != nil {
		return result, fmt.Errorf("failed to get app_key: %v", err)
	}

	title := fmt.Sprintf("sms event:%s", os.Args[1])

	msgBody, err := ioutil.ReadFile(os.Args[2])

	if err != nil {
		return result, fmt.Errorf("failed to read message body file '%s': %v", os.Args[2], err)
	}

	message := PushOverMessage{User: userKey.String(), Token: appKey.String(), Title: title, Message: string(msgBody)}
	result, err = json.Marshal(message)

	if err != nil {
		return result, fmt.Errorf("failed to marshal message: %v", err)
	}

	return result, nil
}

func main() {
	if len(os.Args) != 3 {
		log.Fatalf("this program requires 2 arguments, got:%d", len(os.Args))
	}

	wdConf, err := filepath.Abs("conf.ini")

	if err != nil {
		log.Fatalf("failed to get abs path for working directory file: %v", err)
	}

	exeDir, err := filepath.Abs(filepath.Dir(os.Args[0]))

	if err != nil {
		log.Fatalf("failed to get abs path of exe directory: %v", err)
	}

	confPath := firstExists(filepath.Join(exeDir, "conf.ini"), wdConf)

	if confPath == "" {
		log.Fatalf("failed to find conf.ini file")
	}

	log.Printf("using config file: %s", confPath)

	cfg, err := ini.Load(confPath)

	if err != nil {
		log.Fatalf("fail to read file: %v", err)
	}

	rawMessage, err := pushOverMessage(cfg, confPath)

	if err != nil {
		log.Fatalf("failed to build pushover message: %v", err)
	}

	log.Printf("sending %s", string(rawMessage))

	req, err := http.NewRequest("POST", "https://api.pushover.net/1/messages.json", bytes.NewBuffer(rawMessage))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("failed to preform http post: %v", err)
	}

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.Printf("failed to close body: %v", err)
		}
	}()

	log.Printf("response Status: %s", resp.Status)
	log.Printf("response Headers: %v", resp.Header)
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Fatalf("failed to read response body: %v", err)
	}

	log.Printf("response Body: %s", string(body))

	if resp.StatusCode != 200 {
		log.Fatalf("status other than 200 recieved: %s", resp.Status)
	}

	pushOverResponse := PushOverResponse{}

	err = json.Unmarshal(body, &pushOverResponse)

	if err != nil {
		log.Fatalf("failed to unmarshal response from pushover: %v", err)
	}

	if pushOverResponse.Status != 1 {
		log.Fatalf("status other than 1 returned: %d", pushOverResponse.Status)
	}

}
