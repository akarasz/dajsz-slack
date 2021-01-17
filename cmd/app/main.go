package main

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"text/template"

	"github.com/slack-go/slack"
)

const tpl = `
<!DOCTYPE html>
<html>

<head>
  <meta charset="utf-8">
  <title>Dajsz</title>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <style type="text/css">
    body {
      display: grid;
      grid-template-columns: 100%;
      grid-template-rows: 100%;
      width: 100vw;
      height: 100vh;
      margin: 0;
    }

    div.content {
      grid-column: 1;
      grid-row: 1;
      justify-self: center;
      align-self: center;
    }
  </style>
</head>

<body>
  <div class="content">{{ . }}</div>
</body>
</html>
`

const success = `
<div class="content" style="background-color: rgba(58, 186, 41, .2);padding: .5em 2em;border-radius: .5em;border: 2px solid rgba(58, 186, 41, .3);box-shadow: 0 0 1em rgba(0, 0, 0, 0.1);color: rgba(0,0,0, .8);margin: 5em;">
  <p>
    Dajsz was successfully installed to your workspace, so go over there and play! You can close this window now.
  </p>
</div>
`

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
	http.HandleFunc("/auth/success", successHandler)
	http.HandleFunc("/auth", authHandler)
	http.HandleFunc("/", shortcutHandler)

	http.ListenAndServe(":3000", nil)
}

func authHandler(w http.ResponseWriter, r *http.Request) {
	code, ok := r.URL.Query()["code"]
	if !ok || len(code) != 1 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Print("before getting oauth2 response")
	res, err := slack.GetOAuthV2Response(
		&http.Client{},
		os.Getenv("CLIENT_ID"),
		os.Getenv("CLIENT_SECRET"),
		code[0],
		"https://slack.dajsz.hu/auth/success")
	log.Print("after getting oauth2 response")
	if err != nil {
		logError(w, "failed to get oauth token", err)
		return
	}
	log.Print("got token for ", res.Team.ID, " and it is ", res.AccessToken)

	w.WriteHeader(http.StatusOK)
}

func successHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.New("main").Parse(tpl)
	if err != nil {
		logError(w, "error creating template", err)
		return
	}
	err = t.Execute(w, success)
	if err != nil {
		logError(w, "error while executing on template", err)
		return
	}
	return
}

func shortcutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.Header().Set("Location", "https://slack.com/oauth/v2/authorize?scope=commands&client_id="+os.Getenv("CLIENT_ID"))
		w.WriteHeader(http.StatusFound)
		return
	}

	verifier, err := slack.NewSecretsVerifier(r.Header, os.Getenv("SIGNING_SECRET"))
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
}

func showModal(i *ExtendedInteractionCallback) {
	api := slack.New(os.Getenv("TOKEN"))
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
		Text:         "https://dajsz.hu" + gameID,
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
