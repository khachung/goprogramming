package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/apex/log"
)

type (
	Request struct {
		Object string `json:"object,omitempty"`
		Entry  []struct {
			ID        string      `json:"id,omitempty"`
			Time      int64       `json:"time,omitempty"`
			Messaging []Messaging `json:"messaging,omitempty"`
		} `json:"entry,omitempty"`
	}

	Messaging struct {
		Sender    *User     `json:"sender,omitempty"`
		Recipient *User     `json:"recipient,omitempty"`
		Timestamp int       `json:"timestamp,omitempty"`
		Message   *Message  `json:"message,omitempty"`
		PostBack  *PostBack `json:"postback,omitempty"`
	}

	User struct {
		ID string `json:"id,omitempty"`
	}

	Message struct {
		MID        string      `json:"mid,omitempty"`
		Text       string      `json:"text,omitempty"`
		QuickReply *QuickReply `json:"quick_reply,omitempty"`
		// Attachment []Attachment `json:"attachments,omitempty"`
	}

	QuickReply struct {
		ContentType string `json:"content_type,omitempty"`
		Title       string `json:"title,omitempty"`
		Payload     string `json:"payload"`
	}

	PostBack struct {
		Title   string `json:"title,omitempty"`
		Payload string `json:"payload"`
	}

	ResponseMessage struct {
		MessageType string      `json:"messaging_type"`
		Recipient   *User       `json:"recipient"`
		Message     *ResMessage `json:"message,omitempty"`
		Action      string      `json:"sender_action,omitempty"`
		Tag         string      `json:"tag,omitempty"`
	}

	ResMessage struct {
		// MID  string `json:"mid,omitempty"`
		Text       string       `json:"text,omitempty"`
		QuickReply []QuickReply `json:"quick_replies,omitempty"`
		// Attachment *Attachment  `json:"attachment,omitempty"`
	}
)

type (
	PageProfile struct {
		Greeting       []Greeting       `json:"greeting,omitempty"`
		GetStarted     *GetStarted      `json:"get_started,omitempty"`
		PersistentMenu []PersistentMenu `json:"persistent_menu,omitempty"`
	}

	Greeting struct {
		Locale string `json:"locale,omitempty"`
		Text   string `json:"text,omitempty"`
	}

	GetStarted struct {
		Payload string `json:"payload,omitempty"`
	}

	PersistentMenu struct {
		Locale   string `json:"locale"`
		Composer bool   `json:"composer_input_disabled"`
		CTAs     []CTA  `json:"call_to_actions"`
	}

	CTA struct {
		Type    string `json:"type"`
		Title   string `json:"title"`
		URL     string `json:"url,,omitempty"`
		Payload string `json:"payload"`
		CTAs    []CTA  `json:"call_to_actions,omitempty"`
	}
)

const (
	FBMessageURL    = "https://graph.facebook.com/v3.1/me/messages"
	PageToken       = "EAAH28ZBHCpHsBAHX3hjZC0tdj4pqbLiMTs2tfzlB7z521NwZC7OQ0fI6ZBoh1XKGbMvRjdZCZAKbD2TYdqhnhZCDk1eXAjBZCNJZAIM9rgkaxjfmyONmF6A56UvGNd7ZAXFk8h37FimCq1AnLO5msOfb2pit0j8JHSUPnrUBdwfru04gZDZD"
	MessageResponse = "RESPONSE"
	TypingOn        = "typing_on"
	TypingOff       = "typing_off"
	MarkSeen        = "mark_seen"
)

// Gửi data dạng JSON về server
func sendFBRequest(url string, m interface{}) error {
	// Chuyển struct thành chuỗi byte JSON
	body := new(bytes.Buffer)
	err := json.NewEncoder(body).Encode(&m)
	if err != nil {
		log.Error("sendFBRequest:json.NewEncoder: " + err.Error())
		return err
	}

	// Tạo thông tin http request
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		log.Error("sendFBRequest:http.NewRequest:" + err.Error())
		return err
	}

	// Chọn định dạng gửi là JSON
	req.Header.Add("Content-Type", "application/json")
	// Khai báo token nhận được khi tạo app
	req.URL.RawQuery = "access_token=" + PageToken
	// Tạo đối tượng client với timeout connect 30s
	client := &http.Client{Timeout: time.Second * 30}

	resp, err := client.Do(req)
	if err != nil {
		log.Error("sendFBRequest:client.Do: " + err.Error())
		return err
	}
	defer resp.Body.Close()

	return nil
}

// Gửi tin nhắn kèm trả lời nhanh
func sendTextWithQuickReply(recipient *User, message string, replies []QuickReply) error {
	m := ResponseMessage{
		MessageType: MessageResponse,
		Recipient:   recipient,
		Message: &ResMessage{
			Text:       message,
			QuickReply: replies,
		},
	}
	return sendFBRequest(FBMessageURL, &m)
}

// Gửi tin nhắn văn bản
func sendText(recipient *User, message string) error {
	return sendTextWithQuickReply(recipient, message, nil)
}

// Gửi hành động
func sendAction(recipient *User, action string) error {
	m := ResponseMessage{
		MessageType: MessageResponse,
		Recipient:   recipient,
		Action:      action,
	}
	return sendFBRequest(FBMessageURL, &m)
}

// Đăng ký màn hình chào và menu
func registerGreetingnMenu() bool {
	profile := PageProfile{
		Greeting: []Greeting{
			{
				Locale: "default",
				Text:   "Dịch vụ cung cấp thông tin tỉ giá hối đoái",
			},
		},
		GetStarted: &GetStarted{Payload: GetStartedPB},
		PersistentMenu: []PersistentMenu{
			{
				Locale:   "default",
				Composer: false,
				CTAs: []CTA{
					{
						Type:    "postback",
						Title:   "Tỉ giá hối đoái",
						Payload: RatePB,
					},
				},
			},
		},
	}
	err := sendFBRequest("https://graph.facebook.com/v3.1/me/messenger_profile", &profile)
	if err != nil {
		log.Error("registerGreetingnMenu:" + err.Error())
		return false
	}
	return true
}
