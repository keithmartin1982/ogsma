package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	_ "embed"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

var (
	background         = false
	currentContactName string
	//go:embed config.json
	config          []byte
	contactMessages map[string][]QueueMessage
)

type QueueMessage struct {
	sent time.Time
	msg  string
}

type Config struct {
	Addr     string `json:"addr"`
	KeyStore string `json:"keystore"`
	Endpoint string `json:"endpoint"`
}

type GUI struct {
	app             fyne.App
	window          fyne.Window
	chatOutput      *widget.RichText
	scrollContainer *container.Scroll
	client          *Client
	enc             *Encryption
}

func (g *GUI) loginWindow() {
	g.window.SetTitle("websocket-chat-gui")
	messageLabel := widget.NewLabel("")
	passEntry := widget.NewPasswordEntry()
	passEntry.SetPlaceHolder("Password")
	login := func() {
		g.enc.password = passEntry.Text
		if err := g.enc.loadKeys(); err != nil {
			messageLabel.SetText(fmt.Sprintf("Invalid Password: %v", err))
			return
		}
		g.client.ID = g.enc.keys.ID
		if err := g.client.Connect(); err != nil {
			log.Fatal(err)
		}
		g.contactsWindow()
	}
	passEntry.OnSubmitted = func(s string) {
		login()
	}
	// passEntry.SetText("password1234!") // set password
	loginButton := widget.NewButton("Login", login)
	content := container.NewVBox(
		widget.NewLabel("Please log in"),
		passEntry,
		loginButton,
		messageLabel,
	)
	g.window.SetContent(content)
}

func (g *GUI) contactsWindow() {
	g.window.SetTitle("Contacts")
	content := container.NewVBox(widget.NewLabel("Contacts"))
	for _, contact := range g.enc.keys.Contacts {
		content.Add(widget.NewButton(contact.Username, func() {
			currentContactName = contact.Username
			g.client.targetID = contact.ID
			g.enc.targetPublicKey = contact.PublicKey
			g.chatWindow()
		}))
		content.Add(widget.NewSeparator())
	}
	g.window.SetContent(content)
}

func (g *GUI) chatWindow() {
	g.window.SetTitle("messaging")
	g.chatOutput = widget.NewRichText()
	g.chatOutput.Wrapping = fyne.TextWrapWord
	g.scrollContainer = container.NewVScroll(g.chatOutput)
	msgEntry := widget.NewEntry()
	msgEntry.OnSubmitted = func(s string) {
		if len(s) > 0 {
			targetPublicKeyEncryptedBytes, err := g.enc.publicEncrypt([]byte(msgEntry.Text))
			if err != nil {
				log.Println(err)
				return
			}
			// TODO : resend if fail
			if err := g.client.SendMsg(&Msg{
				ID:        g.client.targetID,
				TimeStamp: time.Now(),
				Message:   targetPublicKeyEncryptedBytes,
				FromID:    g.client.ID,
			}); err != nil {
				log.Println(err)
				g.appendText("Error:", err.Error())
				if err := g.client.Connect(); err != nil {
					g.appendText("Error:", "Failed to reconnect to server")
					g.appendText("Error:", err.Error())
					time.Sleep(5 * time.Second)
					os.Exit(1)
				}
			}
			g.appendText(g.enc.keys.Username, msgEntry.Text)
			msgEntry.SetText("")
		}
	}
	backButton := widget.NewButton("back", func() {
		g.client.targetID = ""
		g.contactsWindow()
	})
	content := container.New(layout.NewBorderLayout(backButton, msgEntry, nil, nil),
		backButton,
		g.scrollContainer,
		msgEntry,
	)
	if val, ok := contactMessages[g.client.targetID]; ok {
		for i := 0; i < len(val); i++ {
			since := time.Now().Sub(val[i].sent).Round(time.Second)
			if since > time.Second*5 {
				g.appendText(fmt.Sprintf("%s %v:", currentContactName, since), val[i].msg)
			} else {
				g.appendText(currentContactName, val[i].msg)
			}

		}
		delete(contactMessages, g.client.targetID)
	}
	g.window.SetContent(content)
}

func (g *GUI) appendText(prefix, content any) {
	go func() {
		fyne.DoAndWait(func() {
			g.chatOutput.AppendMarkdown(fmt.Sprintf("%v: %v", prefix, content))
			g.chatOutput.AppendMarkdown("---")
			g.chatOutput.Refresh()
			g.scrollContainer.ScrollToBottom()
		})
	}()
}

func (g *GUI) lifecycle() {
	// App battery usage unrestricted is required for background websocket connection
	lifecycle := g.app.Lifecycle()
	lifecycle.SetOnExitedForeground(func() {
		background = true
	})
	lifecycle.SetOnEnteredForeground(func() {
		background = false
	})
}

func main() {
	contactMessages = make(map[string][]QueueMessage)
	c := Config{}
	if err := json.Unmarshal(config, &c); err != nil {
		log.Fatalf("Error parsing config file: %v\n", err)
	}
	g := &GUI{
		enc: &Encryption{
			iter:           100000,
			configKeystore: []byte(c.KeyStore),
		},
		client: &Client{
			SelfSigned:  true,
			Addr:        c.Addr,
			wsPath:      c.Endpoint,
			MessageChan: make(chan []byte),
		},
		app: app.New(),
	}
	g.window = g.app.NewWindow("Login")
	platformDo(g)
	g.loginWindow()
	go func() {
		for {
			nm := <-g.client.MessageChan
			nms := Msg{}
			if err := json.Unmarshal(nm, &nms); err != nil {
				log.Printf("error unmarshalling message: %v", err)
			}
			decryptedMessage, err := g.enc.privateDecrypt(nms.Message)
			if err != nil {
				log.Printf("error decrypting message: %v", err)
			}
			since := time.Now().Sub(nms.TimeStamp).Round(time.Second)
			if nms.FromID != g.client.targetID {
				contactMessages[nms.FromID] = append(contactMessages[nms.FromID], QueueMessage{
					sent: nms.TimeStamp,
					msg:  string(decryptedMessage),
				})
			} else {
				if since > time.Second*5 {
					g.appendText(fmt.Sprintf("%s %v:", currentContactName, since), string(decryptedMessage))
				} else {
					g.appendText(currentContactName, string(decryptedMessage))
				}
			}
			if background {
				var rmun string
				for _, contact := range g.enc.keys.Contacts {
					if nms.FromID == contact.ID {
						rmun = contact.Username
						break
					}
				}
				fyne.CurrentApp().SendNotification(&fyne.Notification{
					Title:   fmt.Sprintf("Msg from: %s", rmun),
					Content: fmt.Sprintf("%s", string(decryptedMessage)),
				})
			}
		}
	}()
	g.lifecycle()
	g.window.ShowAndRun()
}
