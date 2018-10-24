package main

import (
	"encoding/json"
	"net/http"

	"github.com/apex/log"
	"github.com/gorilla/mux"
)

func chatbotHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET": // Xác minh webhook
		verifyWebhook(w, r)
	case "POST": // Xử lý sự kiện webhook
		processWebhook(w, r)
	default:
		w.WriteHeader(http.StatusBadRequest)
		log.Errorf("Không hỗ trợ phương thức HTTP %v", r.Method)
	}
}

// Xác minh mã Facebook gửi khớp với mã đã khai
func verifyWebhook(w http.ResponseWriter, r *http.Request) {
	mode := r.URL.Query().Get("hub.mode")
	challenge := r.URL.Query().Get("hub.challenge")
	token := r.URL.Query().Get("hub.verify_token")

	if mode == "subscribe" && token == "GoBot" {
		w.WriteHeader(200)
		w.Write([]byte(challenge))
	} else {
		w.WriteHeader(404)
		w.Write([]byte("Error, wrong validation token"))
	}
}

// Xử lý mọi sự kiện nhận từ Facebook mà app đã đăng ký
func processWebhook(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var req Request
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(404)
		w.Write([]byte("Message not supported"))
		return
	}

	if req.Object == "page" {
		for _, entry := range req.Entry {
			for _, event := range entry.Messaging {
				if event.Message != nil {
					processMessage(&event)
				} else if event.PostBack != nil {
					processPostBack(&event)
				}
			}
		}
		w.WriteHeader(200)
		w.Write([]byte("Got your message"))
	} else {
		w.WriteHeader(404)
		w.Write([]byte("Message not supported"))
	}
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/", chatbotHandler)
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal(err.Error())
	}
}
