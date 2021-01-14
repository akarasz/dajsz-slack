package main

import (
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/slack-go/slack"
)

func main() {
	var (
		oauthToken    string
		signingSecret string
	)

	flag.StringVar(&oauthToken, "token", "YOUR_TOKEN_HERE", "Your Slack app's oauth token")
	flag.StringVar(&signingSecret, "secret", "YOUR_SIGNING_SECRET_HERE", "Your Slack app's signing secret")

	flag.Parse()

	api := slack.New(oauthToken)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		verifier, err := slack.NewSecretsVerifier(r.Header, signingSecret)
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