package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
	
	"github.com/gorilla/websocket"
)

// TODO : only allow user ID's that are in the config file
// TODO : server_config.json

type Server struct {
	endpoint     string
	websockets   map[string]*websocket.Conn
	messageQueue map[string][][]byte
	tlsPort      int
	cert         string
	key          string
	upgrader     websocket.Upgrader
}

type MessageTemplate struct {
	ID  string `json:"id"`
	Msg []byte `json:"msg"`
}

func (s *Server) oc() func(r *http.Request) bool {
	return func(r *http.Request) bool {
		return true
	}
}

func (s *Server) start() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://youtu.be/dQw4w9WgXcQ", http.StatusMovedPermanently) // ROFL
	})
	http.HandleFunc(fmt.Sprintf("/%s", s.endpoint), func(w http.ResponseWriter, r *http.Request) {
		var currentUserID string
		s.upgrader.CheckOrigin = s.oc()
		c, err := s.upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("upgrade to websocket conn:", err)
			return
		}
		websocketTimeout := time.Now()
		c.SetPingHandler(func(m string) error {
			websocketTimeout = time.Now()
			if err := c.WriteMessage(websocket.PongMessage, []byte("pong")); err != nil {
				return errors.New("websocket pong: " + err.Error())
			}
			return nil
		})
		go func() {
			for {
				if time.Now().Sub(websocketTimeout) > time.Second*3 {
					log.Printf("removing websocket session for %s\n", currentUserID)
					delete(s.websockets, currentUserID)
					return
				}
			}
		}()
		pmt, pm, err := c.ReadMessage()
		if err != nil {
			log.Printf("Error reading init message: %v\n", err)
			return
		}
		if pmt == websocket.BinaryMessage {
			prs := struct {
				ID string `json:"id"`
			}{}
			if err := json.Unmarshal(pm, &prs); err != nil {
				log.Printf("Error parsing init message: %v\n", err)
			}
			if len(prs.ID) != 64 {
				return
			}
			currentUserID = prs.ID
			s.websockets[prs.ID] = c
			log.Printf("User  %s Connected from %s\n", prs.ID, r.RemoteAddr)
		} else {
			log.Printf("Error parsing init message: %v\n", string(pm))
			return
		}
		if val, ok := s.messageQueue[currentUserID]; ok {
			for i := 0; i < len(val); i++ {
				if err := s.websockets[currentUserID].WriteMessage(websocket.TextMessage, val[i]); err != nil {
					log.Printf("Error writing message: %v\n", err)
					return
				}
			}
			delete(s.messageQueue, currentUserID)
		}
		for {
			messageType, message, err := c.ReadMessage()
			if err != nil {
				log.Printf("Error reading message: %v\n", err)
				log.Printf("removing websocket session for %s\n", currentUserID)
				delete(s.websockets, currentUserID)
				return
			}
			switch messageType {
			case websocket.TextMessage:
				mt := &MessageTemplate{}
				if err := json.Unmarshal(message, mt); err != nil {
					log.Printf("Error parsing message: %v\n", err)
					return
				}
				if value, ok := s.websockets[mt.ID]; ok {
					if err := value.WriteMessage(messageType, message); err != nil {
						log.Printf("Error writing message: %v\n", err)
						return
					}
				} else {
					s.messageQueue[mt.ID] = append(s.messageQueue[mt.ID], message)
				}
			default:
				fmt.Printf("Unused message type: %v\n", messageType)
				continue
			}
		}
	})
	if err := http.ListenAndServeTLS(fmt.Sprintf(":%d", s.tlsPort), s.cert, s.key, nil); err != nil {
		log.Printf("ListenAndServeTLS: %v\n", err)
	}
}

func main() {
	s := &Server{}
	s.websockets = make(map[string]*websocket.Conn)
	s.messageQueue = make(map[string][][]byte)
	flag.StringVar(&s.endpoint, "endpoint", "ws", "websocket endpoint")
	flag.StringVar(&s.cert, "cert", "server.crt", "tls cert file")
	flag.StringVar(&s.key, "key", "server.key", "tls key file")
	flag.IntVar(&s.tlsPort, "port", 8443, "https port")
	flag.Parse()
	s.start()
}
