package main

import (
	"GoInspect/csgo"
	"GoInspect/gsbot"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/Philipp15b/go-steam/v3"
	"github.com/Philipp15b/go-steam/v3/protocol/steamlang"
	"github.com/Philipp15b/go-steam/v3/totp"
)

func startServer(addr string, csgoClient *csgo.CSGO, info *sync.Map) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		link := r.URL.Query().Get("link")
		re := regexp.MustCompile(`([SM])(\d+)A(\d+)D(\d+)$`)
		matches := re.FindStringSubmatch(link)
		if len(matches) != 5 {
			return
		}
		// don't know why some links just don't work
		csgoClient.InspectItem(matches[1], matches[2], matches[3], matches[4])

		waitTimes := 30
		for waitTimes > 0 {
			data, ok := info.LoadAndDelete(matches[3])
			if ok {
				jsonData, _ := json.Marshal(data)
				w.WriteHeader(http.StatusOK)
				w.Header().Set("Content-Type", "application/json")
				w.Write(jsonData)
				return
			}
			time.Sleep(time.Second / 2)
			waitTimes--
		}
		outData := map[string]bool{"timeout": true}
		jsonData, _ := json.Marshal(outData)
		w.WriteHeader(http.StatusGatewayTimeout)
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonData)
	})
	http.ListenAndServe(addr, nil)
}

func main() {
	// todo: add info to iteminfo.
	// todo: https://github.com/Jleagle/steam-go/tree/master/steamvdf
	// todo: load user, password, sharedsecret from config file
	// todo: multi account support
	details := &steam.LogOnDetails{
		Username:               os.Getenv("STEAM_USERNAME"),
		Password:               os.Getenv("STEAM_PASSWORD"),
		ShouldRememberPassword: true,
	}
	details.TwoFactorCode, _ = totp.NewTotp(os.Getenv("STEAM_SHAREDSECRET")).GenerateCode()

	bot := gsbot.Default()
	client := bot.Client

	auth := gsbot.NewAuth(bot, details, "sentry")

	serverList := gsbot.NewServerList(bot)
	serverList.Connect()

	signal := make(chan bool)
	var info sync.Map

	go func() {
		for event := range client.Events() {
			auth.HandleEvent(event)
			serverList.HandleEvent(event)
			switch e := event.(type) {
			case error:
				fmt.Println(e)
				serverList.Connect()
			case *steam.LoggedOnEvent:
				client.Social.SetPersonaState(steamlang.EPersonaState_Online)
			case *steam.LoginKeyEvent:
				// steam logged in
				signal <- true
			case *csgo.ItemInfo:
				info.Store(csgo.DeParam(e.ItemId), e)
			case *csgo.ClientReady:
				// csgo logged in
				signal <- true
			}
		}
	}()

	// Logged in to Steam
	<-signal

	print("Initializing CSGO client...\n")
	csgoClient := csgo.New(client)

	// Logged in to CSGO
	<-signal

	print("Listening on http://localhost:8080/")
	startServer("127.0.0.1:8080", csgoClient, &info)
}
