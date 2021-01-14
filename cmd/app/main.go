package main

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/slack-go/slack"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		verifier, err := slack.NewSecretsVerifier(r.Header, os.Getenv("secret"))
		if err != nil {
			logError(w, "unable to create secrets verifier", err)
			return
		}

		r.Body = ioutil.NopCloser(io.TeeReader(r.Body, &verifier))
		s, err := slack.SlashCommandParse(r)
		if err != nil {
			logError(w, "unable to parse command", err)
			return
		}

		if err = verifier.Ensure(); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		switch s.Command {
		case "/dajsz":
			go sendDajszLink(s.ResponseURL)
			w.WriteHeader(http.StatusOK)
			return
		default:
			logError(w, "invalid command", s.Command)
			return
		}
	})

	http.ListenAndServe(":3000", nil)
}

func sendDajszLink(responseURL string) {
	res, err := http.Post("https://api.dajsz.hu/", "", nil)
	if err != nil {
		sendFail(responseURL)
		return
	}
	if res.StatusCode != http.StatusCreated {
		sendFail(responseURL)
		return
	}
	gameIDs, ok := res.Header["Location"]
	if !ok {
		sendFail(responseURL)
		return
	}
	if len(gameIDs) != 1 {
		sendFail(responseURL)
		return
	}

	sendSuccess(responseURL, gameIDs[0])
}

func sendSuccess(responseURL, gameID string) {
	send(responseURL, &slack.Msg{
		Text:         ":game_die: Ic dajsz tajm! https://dajsz.hu/#" + gameID,
		ResponseType: "in_channel",
	})
}

func sendFail(responseURL string) {
	send(responseURL, &slack.Msg{
		Text: "Something went wrong. Try again?",
	})
}

func send(responseURL string, message *slack.Msg) {
	body, err := json.Marshal(message)
	if err != nil {
		return
	}

	http.Post(responseURL, "application/json", bytes.NewReader(body))
}

func logError(w http.ResponseWriter, msg string, details interface{}) {
	log.Printf("%s: %v", msg, details)
	w.WriteHeader(http.StatusInternalServerError)
}
