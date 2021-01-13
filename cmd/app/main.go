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

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		verifier, err := slack.NewSecretsVerifier(r.Header, signingSecret)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		r.Body = ioutil.NopCloser(io.TeeReader(r.Body, &verifier))
		s, err := slack.SlashCommandParse(r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err = verifier.Ensure(); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		switch s.Command {
		case "/dajsz":
			api := slack.New(oauthToken)

			channelID, timestamp, err := api.PostMessage(
				s.ChannelID,
				slack.MsgOptionText("You shoud _definitely_ play a game :game_die:", false),
				slack.MsgOptionAsUser(true),
			)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			log.Printf("Message successfully sent to channel %s at %s", channelID, timestamp)
			w.WriteHeader(http.StatusOK)
			return
		default:
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})

	http.ListenAndServe(":3000", nil)
}
