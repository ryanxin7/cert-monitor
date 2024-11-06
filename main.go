package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// 监控的URL列表
var urlsToCheck = []string{
	"https://ceamg.com",
	"https://file.ceamg.com:8884",
	"https://www.cnenergynews.cn",
	"https://file.cnenergynews.cn:8880",

}

// 钉钉Webhook URL
const dingWebhook = "https://oapi.dingtalk.com/robot/send?access_token=9db75a93f3504f66e8417aaac01358be817644fa3549172c5be34bde3de9ad27"

// 提前提醒的天数
const daysBeforeExpiration = 3

// 定时任务，每天早上10点检查
const checkHour = 10

// 钉钉消息结构体
type DingMessage struct {
	MsgType string `json:"msgtype"`
	Text    struct {
		Content string `json:"content"`
	} `json:"text"`
}

// 获取HTTPS证书到期日期
func getCertExpiry(url string) (time.Time, error) {
	resp, err := http.Get(url)
	if err != nil {
		return time.Time{}, err
	}
	defer resp.Body.Close()

	if resp.TLS == nil || len(resp.TLS.PeerCertificates) == 0 {
		return time.Time{}, fmt.Errorf("no TLS certificate found")
	}

	return resp.TLS.PeerCertificates[0].NotAfter, nil
}

// 发送钉钉提醒消息
func sendDingAlert(message string) error {
	//初始化了一个 DingMessage 实例
	msg := DingMessage{MsgType: "text"}
	//设置消息内容
	msg.Text.Content = message


	//消息 msg 转换为 JSON 格式
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	//将字节数组包装成 *bytes.Buffer 缓冲区对象，该对象实现了 io.Reader 接口。
	resp, err := http.Post(dingWebhook, "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Printf("Failed to send request: %v\n", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("Request failed with status code: %d\n", resp.StatusCode)
		return fmt.Errorf("failed to send ding alert, status code: %d", resp.StatusCode)
	}
	log.Println("Ding alert sent successfully")
	return nil
}

// 检查URL证书的到期日期
func checkCertificates() {
	for _, url := range urlsToCheck {
		expiryDate, err := getCertExpiry(url)
		if err != nil {
			log.Printf("Failed to check certificate for %s: %v\n", url, err)
			continue
		}

		daysLeft := int(expiryDate.Sub(time.Now()).Hours() / 24)
		if daysLeft <= daysBeforeExpiration {
			message := fmt.Sprintf("HTTPS证书提醒: %s 的证书将在 %d 天后过期（到期日: %s）", url, daysLeft, expiryDate.Format("2006-01-02"))
			log.Println(message)
			//如果发送成功则继续，否则返回一个错误。
			if err := sendDingAlert(message); err != nil {
				log.Printf("Failed to send Ding alert: %v\n", err)
			}
		} else {
			log.Printf("%s 的证书还有 %d 天过期\n", url, daysLeft)
		}
	}
}

func main() {
	// 每天定时检查
	ticker := time.NewTicker(1 * time.Hour)
	//匿名函数
	go func() {
		for {
			now := time.Now()
			if now.Hour() == checkHour {
				//checkHour：这是预设的检查小时数
				checkCertificates()
			}
			<-ticker.C
			//等待从ticker.C 通道接收到信号
			//由于ticker每 24 小时触发一次，所以这个等待过程会让循环每24小时运行一次。
		}

	}()
	//() 的作用是用来立即执行匿名函数

	// 首次运行时立即检查
	checkCertificates()

	// 防止程序退出
	select {}
}
