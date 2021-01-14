package main

import (
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/slack-go/slack"
)

func main() {
	api := slack.New(os.Getenv("token"))

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
			res, err := http.Post("https://api.dajsz.hu/", "", nil)

			if err != nil {
				logError(w, "failed to post request", err)
				return
			}
			if res.StatusCode != http.StatusCreated {
				logError(w, "wrong return from api", res.StatusCode)
				return
			}

			gameIDs, ok := res.Header["Location"]
			if !ok {
				logError(w, "no location header in api response", nil)
				return
			}
			if len(gameIDs) != 1 {
				logError(w, "invalid location header in api response", gameIDs)
				return
			}

			_, _, err = api.PostMessage(
				s.ChannelID,
				slack.MsgOptionText("Join a friendly game of yahtzee: https://dajsz.hu/#" + gameIDs[0], false),
				slack.MsgOptionAsUser(true),
			)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			return
		default:
			logError(w, "invalid command", s.Command)
			return
		}
	})

	http.ListenAndServe(":3000", nil)
}

func logError(w http.ResponseWriter, msg string, details interface{}) {
	log.Printf("%s: %v", msg, details)
	w.WriteHeader(http.StatusInternalServerError)
}