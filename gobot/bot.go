package main

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/apex/log"
)

type (
	ExchangeRate struct {
		DateTime string   `xml:"DateTime"`
		Exrate   []Exrate `xml:"Exrate"`
		Source   string   `xml:"Source"`
	}

	Exrate struct {
		CurrencyCode string `xml:"CurrencyCode,attr"`
		CurrencyName string `xml:"CurrencyName,attr"`
		Buy          string `xml:"Buy,attr"`
		Transfer     string `xml:"Transfer,attr"`
		Sell         string `xml:"Sell,attr"`
	}
)

const (
	GetStartedPB = "GetStarted"
	RatePB       = "rate"
)

var (
	// Lưu thông tin ngoại tệ lấy được
	exRateList *ExchangeRate
	// Lưu nhóm ngoại tệ đang hiển thị của từng người dùng
	exRateGroupMap = make(map[string]int)
)

func processMessage(event *Messaging) {
	// Gửi hành động đã xem và đang trả lời
	sendAction(event.Sender, MarkSeen)
	sendAction(event.Sender, TypingOn)

	// Xử lý khi người dùng chọn trả lời nhanh
	if event.Message.QuickReply != nil {
		processQuickReply(event)
		return
	}
	// Xử lý khi người dùng gửi văn bản
	text := strings.ToLower(strings.TrimSpace(event.Message.Text))
	if text == "rate" {
		// Lưu nhóm ngoại tệ xem hiện tại
		exRateGroupMap[event.Sender.ID] = 1
		// Gửi danh sách ngoại tệ
		sendExchangeRateList(event.Sender)
	} else {
		// Gửi chuỗi nhận được sau khi chuyển sang chữ hoa
		sendText(event.Sender, strings.ToUpper(event.Message.Text))
	}
	// Gửi hành động đã trả lời xong
	sendAction(event.Sender, TypingOff)
}

func processQuickReply(event *Messaging) {
	recipient := event.Sender
	exRateGroup := exRateGroupMap[event.Sender.ID]
	switch event.Message.QuickReply.Payload {
	case "Next": // Trường hợp người dùng chọn "Xem tiếp"
		var i int
		// Kiểm tra nếu đã xem xong danh sách thì quay lại
		if exRateGroup*10 >= len(exRateList.Exrate) {
			exRateGroup = 1
		} else {
			exRateGroup++
		}
		exRateGroupMap[event.Sender.ID] = exRateGroup
		quickRep := []QuickReply{}
		// Mỗi lần hiển thị gồm 10 ngoại tệ
		for i = 10 * (exRateGroup - 1); i < 10*exRateGroup && i < len(exRateList.Exrate); i++ {
			exrate := exRateList.Exrate[i]
			quickRep = append(quickRep, QuickReply{ContentType: "text", Title: exrate.CurrencyName, Payload: exrate.CurrencyCode})
		}
		// Thêm nút "Xem tiếp"
		quickRep = append(quickRep, QuickReply{ContentType: "text", Title: "Xem tiếp", Payload: "Next"})
		sendTextWithQuickReply(recipient, "GoBot cung cấp chức năng xem tỉ giá giữa các ngoại tệ và đồng Việt Nam.\nMời bạn chọn ngoại tệ:", quickRep)
	default: // Trường hợp người dùng chọn 1 nút trả lời nhanh
		var exRate Exrate
		// Kiểm tra coi payload nhận được khớp với item nào
		for i := 10 * (exRateGroup - 1); i < 10*exRateGroup && i < len(exRateList.Exrate); i++ {
			if exRateList.Exrate[i].CurrencyCode == event.Message.QuickReply.Payload {
				exRate = exRateList.Exrate[i]
				break
			}
		}
		// Không tìm thấy item nào khớp
		if len(exRate.CurrencyCode) == 0 {
			sendText(recipient, "Không có thông tin về ngoại tệ này")
			return
		}
		// Trả về thông tin tìm được
		sendText(recipient, fmt.Sprintf("%s-VND\nGiá mua: %sđ\nGiá bán: %sđ\nGiá chuyển khoản: %sđ", exRate.CurrencyCode, exRate.Buy, exRate.Sell, exRate.Transfer))
	}
}

func sendExchangeRateList(recipient *User) {
	var (
		ok          bool
		i           int
		exRateGroup = exRateGroupMap[recipient.ID]
	)
	// Lấy danh sách ngoại tệ và lưu vào biến toàn cục exRateList
	exRateList, ok = getExchangeRateVCB()
	if !ok {
		sendText(recipient, "Có lỗi trong quá trình xử lý. Bạn vui lòng thử lại sau bằng cách gửi 'rate' cho tôi nhé. Cảm ơn!")
		return
	}
	quickRep := []QuickReply{}
	// Lấy nhóm 10 ngoại tệ
	for i = 10 * (exRateGroup - 1); i < 10*exRateGroup && i < len(exRateList.Exrate); i++ {
		exrate := exRateList.Exrate[i]
		quickRep = append(quickRep, QuickReply{ContentType: "text", Title: exrate.CurrencyName, Payload: exrate.CurrencyCode})
	}
	quickRep = append(quickRep, QuickReply{ContentType: "text", Title: "Xem tiếp", Payload: "Next"})
	sendTextWithQuickReply(recipient, "GoBot cung cấp chức năng xem tỉ giá giữa các ngoại tệ và đồng Việt Nam.\nMời bạn chọn ngoại tệ:", quickRep)
}

func getExchangeRateVCB() (*ExchangeRate, bool) {
	var exrate ExchangeRate

	req, err := http.NewRequest("GET", "http://www.vietcombank.com.vn/ExchangeRates/ExrateXML.aspx", nil)
	if err != nil {
		log.Errorf("getExchangeRateVCB: NewRequest: %s", err.Error())
		return &exrate, false
	}

	client := &http.Client{Timeout: time.Second * 30}
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("getExchangeRateVCB: client.Do: %s", err.Error())
		return &exrate, false
	}
	defer resp.Body.Close()

	err = xml.NewDecoder(resp.Body).Decode(&exrate)
	if err != nil {
		log.Errorf("getExchangeRateVCB: xml.NewDecoder: %s", err.Error())
		return &exrate, false
	}

	sort.Slice(exrate.Exrate, func(i, j int) bool {
		return exrate.Exrate[i].CurrencyName < exrate.Exrate[j].CurrencyName
	})
	return &exrate, true
}

func processPostBack(event *Messaging) {
	// Gửi hành động đã xem và đang trả lời
	sendAction(event.Sender, MarkSeen)
	sendAction(event.Sender, TypingOn)

	switch event.PostBack.Payload {
	case GetStartedPB, RatePB:
		exRateGroupMap[event.Sender.ID] = 1
		sendExchangeRateList(event.Sender)
	}
	// Gửi hành động đã trả lời xong
	sendAction(event.Sender, TypingOff)
}
