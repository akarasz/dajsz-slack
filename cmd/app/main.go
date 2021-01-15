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

type ResponseURL struct {
	BlockID     string `json:"block_id"`
	ActionID    string `json:"action_id"`
	ChannelID   string `json:"channel_id"`
	ResponseURL string `json:"response_url"`
}

type ExtendedInteractionCallback struct {
	CallbackID   string        `json:"callback_id"`
	TriggerID    string        `json:"trigger_id"`
	ResponseURLs []ResponseURL `json:"response_urls"`

	// slack.InteractionCallback
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		verifier, err := slack.NewSecretsVerifier(r.Header, os.Getenv("secret"))
		if err != nil {
			logError(w, "unable to create secrets verifier", err)
			return
		}

		r.Body = ioutil.NopCloser(io.TeeReader(r.Body, &verifier))
		var i ExtendedInteractionCallback
		err = json.Unmarshal([]byte(r.FormValue("payload")), &i)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if i.CallbackID == "yahtzee" {
			go showModal(&i)
		} else {
			go sendDajszLink(i.ResponseURLs[0].ResponseURL)
		}
		w.WriteHeader(http.StatusOK)
		return
	})

	http.ListenAndServe(":3000", nil)
}

func showModal(i *ExtendedInteractionCallback) {
	api := slack.New(os.Getenv("token"))
	_, err := api.OpenView(
		i.TriggerID,
		slack.ModalViewRequest{
			Title: &slack.TextBlockObject{
				Type: "plain_text",
				Text: "Dajsz",
			},
			Submit: &slack.TextBlockObject{
				Type: "plain_text",
				Text: "Share link",
			},
			Blocks: slack.Blocks{
				BlockSet: []slack.Block{
					slack.InputBlock{
						BlockID:  "share",
						Type:     "input",
						Optional: false,
						Label: &slack.TextBlockObject{
							Type: "plain_text",
							Text: "Where to share the link of the game?",
						},
						Element: slack.SelectBlockElement{
							ActionID:                     "my_action_id",
							Type:                         "conversations_select",
							DefaultToCurrentConversation: true,
							ResponseURLEnabled:           true,
						},
					},
				},
			},
			Type: "modal",
		})
	if err != nil {
		log.Printf(err.Error())
		return
	}
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