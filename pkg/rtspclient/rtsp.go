package rtspclient

import (
	"crypto/md5"
	"encoding/hex"
	"strings"
	"fmt"
	"net"
	"strconv"
	"log"
	"time"
	"io"
	"net/url"
	b64 "encoding/base64"
)

type RtspClient struct {
	socket   net.Conn
	outgoing chan []byte 
	signals  chan bool   
	host     string      
	port     string      
	uri      string      
	auth     bool        
	login    string
	password string   
	session  string   
	responce string   
	bauth    string   
	track    []string 
	cseq     int      
	videow   int
	videoh   int
}


func RtspClientNew() *RtspClient {
	Obj := &RtspClient{
		cseq:     1,                         
		signals:  make(chan bool, 1),        
		outgoing: make(chan []byte, 100000), 
	}
	return Obj
}


func (this *RtspClient) Client(rtsp_url string) (bool, string) {
	
	if !this.ParseUrl(rtsp_url) {
		return false, "Не верный url"
	}
	
	if !this.Connect() {
		return false, "Не возможно подключиться"
	}
	
	
	if !this.Write("OPTIONS " + this.uri + " RTSP/1.0\r\nCSeq: " + strconv.Itoa(this.cseq) + "\r\n\r\n") {
		return false, "Не возможно отправить сообщение OPTIONS"
	}
	
	if status, message := this.Read(); !status {
		return false, "Не возможно прочитать ответ OPTIONS соединение потеряно"
	} else if status && strings.Contains(message, "Digest") {
		if !this.AuthDigest("OPTIONS", message) {
			return false, "Требуеться авторизация Digest"
		}
	} else if status && strings.Contains(message, "Basic") {
		if !this.AuthBasic("OPTIONS", message) {
			return false, "Требуеться авторизация Basic"
		}
	} else if !strings.Contains(message, "200") {
		return false, "Ошибка OPTIONS not status code 200 OK " + message
	}

	
	log.Println("DESCRIBE " + this.uri + " RTSP/1.0\r\nCSeq: " + strconv.Itoa(this.cseq) + this.bauth + "\r\n\r\n")
	if !this.Write("DESCRIBE " + this.uri + " RTSP/1.0\r\nCSeq: " + strconv.Itoa(this.cseq) + this.bauth + "\r\n\r\n") {
		return false, "Не возможно отправть запрос DESCRIBE"
	}
	if status, message := this.Read(); !status {
		return false, "Не возможно прочитать ответ DESCRIBE соединение потеряно ?"
	} else if status && strings.Contains(message, "Digest") {
		if !this.AuthDigest("DESCRIBE", message) {
			return false, "Требуеться авторизация Digest"
		}
	} else if status && strings.Contains(message, "Basic") {
		if !this.AuthBasic("DESCRIBE", message) {
			return false, "Требуеться авторизация Basic"
		}
	} else if !strings.Contains(message, "200") {
		return false, "Ошибка DESCRIBE not status code 200 OK " + message
	} else {
		log.Println(message)
		this.track = this.ParseMedia(message)

	}
	if len(this.track) == 0 {
		return false, "Ошибка track not found "
	}
	
	log.Println("SETUP " + this.uri + "/" + this.track[0] + " RTSP/1.0\r\nCSeq: " + strconv.Itoa(this.cseq) + "\r\nTransport: RTP/AVP/TCP;unicast;interleaved=0-1" + this.bauth + "\r\n\r\n")
	if !this.Write("SETUP " + this.uri + "/" + this.track[0] + " RTSP/1.0\r\nCSeq: " + strconv.Itoa(this.cseq) + "\r\nTransport: RTP/AVP/TCP;unicast;interleaved=0-1" + this.bauth + "\r\n\r\n") {
		return false, ""
	}
	if status, message := this.Read(); !status {
		return false, "Не возможно прочитать ответ SETUP соединение потеряно"

	} else if !strings.Contains(message, "200") {
		if strings.Contains(message, "401") {
			str := this.AuthDigest_Only("SETUP", message)
			if !this.Write("SETUP " + this.uri + "/" + this.track[0] + " RTSP/1.0\r\nCSeq: " + strconv.Itoa(this.cseq) + "\r\nTransport: RTP/AVP/TCP;unicast;interleaved=0-1" + this.bauth + str + "\r\n\r\n") {
				return false, ""
			}
			if status, message := this.Read(); !status {
				return false, "Не возможно прочитать ответ SETUP соединение потеряно"

			} else if !strings.Contains(message, "200") {

				return false, "Ошибка SETUP not status code 200 OK " + message

			} else {
				this.session = ParseSession(message)
			}
		} else {
			return false, "Ошибка SETUP not status code 200 OK " + message
		}
	} else {
		log.Println(message)
		this.session = ParseSession(message)
		log.Println(this.session)
	}
	if len(this.track) > 1 {

		if !this.Write("SETUP " + this.uri + "/" + this.track[1] + " RTSP/1.0\r\nCSeq: " + strconv.Itoa(this.cseq) + "\r\nTransport: RTP/AVP/TCP;unicast;interleaved=2-3" + "\r\nSession: " + this.session + this.bauth + "\r\n\r\n") {
			return false, ""
		}
		if status, message := this.Read(); !status {
			return false, "Не возможно прочитать ответ SETUP Audio соединение потеряно"

		} else if !strings.Contains(message, "200") {
			if strings.Contains(message, "401") {
				str := this.AuthDigest_Only("SETUP", message)
				if !this.Write("SETUP " + this.uri + "/" + this.track[1] + " RTSP/1.0\r\nCSeq: " + strconv.Itoa(this.cseq) + "\r\nTransport: RTP/AVP/TCP;unicast;interleaved=2-3" + this.bauth + str + "\r\n\r\n") {
					return false, ""
				}
				if status, message := this.Read(); !status {
					return false, "Не возможно прочитать ответ SETUP Audio соединение потеряно"

				} else if !strings.Contains(message, "200") {

					return false, "Ошибка SETUP not status code 200 OK " + message

				} else {
					log.Println(message)
					this.session = ParseSession(message)
				}
			} else {
				return false, "Ошибка SETUP not status code 200 OK " + message
			}
		} else {
			log.Println(message)
			this.session = ParseSession(message)
		}
	}

	
	log.Println("PLAY " + this.uri + " RTSP/1.0\r\nCSeq: " + strconv.Itoa(this.cseq) + "\r\nSession: " + this.session + this.bauth + "\r\n\r\n")
	if !this.Write("PLAY " + this.uri + " RTSP/1.0\r\nCSeq: " + strconv.Itoa(this.cseq) + "\r\nSession: " + this.session + this.bauth + "\r\n\r\n") {
		return false, ""
	}
	if status, message := this.Read(); !status {
		return false, "Не возможно прочитать ответ PLAY соединение потеряно"

	} else if !strings.Contains(message, "200") {
		
		if strings.Contains(message, "401") {
			str := this.AuthDigest_Only("PLAY", message)
			if !this.Write("PLAY " + this.uri + " RTSP/1.0\r\nCSeq: " + strconv.Itoa(this.cseq) + "\r\nSession: " + this.session + this.bauth + str + "\r\n\r\n") {
				return false, ""
			}
			if status, message := this.Read(); !status {
				return false, "Не возможно прочитать ответ PLAY соединение потеряно"

			} else if !strings.Contains(message, "200") {

				return false, "Ошибка PLAY not status code 200 OK " + message

			} else {
				
				log.Print(message)
				go this.RtspRtpLoop()
				return true, "ok"
			}
		} else {
			return false, "Ошибка PLAY not status code 200 OK " + message
		}
	} else {
		log.Print(message)
		go this.RtspRtpLoop()
		return true, "ok"
	}
	return false, "other error"
}

func (this *RtspClient) RtspRtpLoop() {
	defer func() {
		this.signals <- true
	}()
	header := make([]byte, 4)
	payload := make([]byte, 4096)
	
	sync_b := make([]byte, 1)
	timer := time.Now()
	for {
		if int(time.Now().Sub(timer).Seconds()) > 50 {
			if !this.Write("OPTIONS " + this.uri + " RTSP/1.0\r\nCSeq: " + strconv.Itoa(this.cseq) + "\r\nSession: " + this.session + this.bauth + "\r\n\r\n") {
				return
			}
			timer = time.Now()
		}
		this.socket.SetDeadline(time.Now().Add(50 * time.Second))
		
		if n, err := io.ReadFull(this.socket, header); err != nil || n != 4 {
			
			return
		}
		
		if header[0] != 36 {
			
			for {
				
				if n, err := io.ReadFull(this.socket, sync_b); err != nil && n != 1 {
					return
				} else if sync_b[0] == 36 {
					header[0] = 36
					if n, err := io.ReadFull(this.socket, header[1:]); err != nil && n == 3 {
						return
					}
					break
				}
			}
		}

		payloadLen := (int)(header[2])<<8 + (int)(header[3])
		
		if payloadLen > 4096 || payloadLen < 12 {
			log.Println("desync", this.uri, payloadLen)
			return
		}
		if n, err := io.ReadFull(this.socket, payload[:payloadLen]); err != nil || n != payloadLen {
			return
		} else {
			this.outgoing <- append(header, payload[:n]...)
		}
	}

}


func (this *RtspClient) SendBufer(bufer []byte) {
	
	payload := make([]byte, 4096)
	for {
		if len(bufer) < 4 {
			log.Fatal("bufer small")
		}
		dataLength := (int)(bufer[2])<<8 + (int)(bufer[3])
		if dataLength > len(bufer)+4 {
			if n, err := io.ReadFull(this.socket, payload[:dataLength-len(bufer)+4]); err != nil {
				return
			} else {
				this.outgoing <- append(bufer, payload[:n]...)
				return
			}

		} else {
			this.outgoing <- bufer[:dataLength+4]
			bufer = bufer[dataLength+4:]
		}
	}
}
func (this *RtspClient) Connect() bool {
	d := &net.Dialer{Timeout: 3 * time.Second}
	conn, err := d.Dial("tcp", this.host+":"+this.port)
	if err != nil {
		return false
	}
	this.socket = conn
	return true
}
func (this *RtspClient) Write(message string) bool {
	this.cseq += 1
	if _, e := this.socket.Write([]byte(message)); e != nil {
		return false
	}
	return true
}
func (this *RtspClient) Read() (bool, string) {
	buffer := make([]byte, 4096)
	if nb, err := this.socket.Read(buffer); err != nil || nb <= 0 {
		log.Println("socket read failed", err)
		return false, ""
	} else {
		return true, string(buffer[:nb])
	}
}
func (this *RtspClient) AuthBasic(phase string, message string) bool {
	this.bauth = "\r\nAuthorization: Basic " + b64.StdEncoding.EncodeToString([]byte(this.login+":"+this.password))
	if !this.Write(phase + " " + this.uri + " RTSP/1.0\r\nCSeq: " + strconv.Itoa(this.cseq) + this.bauth + "\r\n\r\n") {
		return false
	}
	if status, message := this.Read(); status && strings.Contains(message, "200") {
		this.track = ParseMedia(message)
		return true
	}
	return false
}
func (this *RtspClient) AuthDigest(phase string, message string) bool {
	nonce := ParseDirective(message, "nonce")
	realm := ParseDirective(message, "realm")
	hs1 := GetMD5Hash(this.login + ":" + realm + ":" + this.password)
	hs2 := GetMD5Hash(phase + ":" + this.uri)
	responce := GetMD5Hash(hs1 + ":" + nonce + ":" + hs2)
	dauth := "\r\n" + `Authorization: Digest username="` + this.login + `", realm="` + realm + `", nonce="` + nonce + `", uri="` + this.uri + `", response="` + responce + `"`
	if !this.Write(phase + " " + this.uri + " RTSP/1.0\r\nCSeq: " + strconv.Itoa(this.cseq) + dauth + "\r\n\r\n") {
		return false
	}
	if status, message := this.Read(); status && strings.Contains(message, "200") {
		this.track = ParseMedia(message)
		return true
	}
	return false
}
func (this *RtspClient) AuthDigest_Only(phase string, message string) string {
	nonce := ParseDirective(message, "nonce")
	realm := ParseDirective(message, "realm")
	hs1 := GetMD5Hash(this.login + ":" + realm + ":" + this.password)
	hs2 := GetMD5Hash(phase + ":" + this.uri)
	responce := GetMD5Hash(hs1 + ":" + nonce + ":" + hs2)
	dauth := "\r\n" + `Authorization: Digest username="` + this.login + `", realm="` + realm + `", nonce="` + nonce + `", uri="` + this.uri + `", response="` + responce + `"`
	return dauth
}
func (this *RtspClient) ParseUrl(rtsp_url string) bool {

	u, err := url.Parse(rtsp_url)
	if err != nil {
		return false
	}
	phost := strings.Split(u.Host, ":")
	this.host = phost[0]
	if len(phost) == 2 {
		this.port = phost[1]
	} else {
		this.port = "554"
	}
	this.login = u.User.Username()
	this.password, this.auth = u.User.Password()
	if u.RawQuery != "" {
		this.uri = "rtsp://" + this.host + ":" + this.port + u.Path + "?" + string(u.RawQuery)
	} else {
		this.uri = "rtsp://" + this.host + ":" + this.port + u.Path
	}
	return true
}
func (this *RtspClient) Close() {
	if this.socket != nil {
		this.socket.Close()
	}
}
func ParseDirective(header, name string) string {
	index := strings.Index(header, name)
	if index == -1 {
		return ""
	}
	start := 1 + index + strings.Index(header[index:], `"`)
	end := start + strings.Index(header[start:], `"`)
	return strings.TrimSpace(header[start:end])
}

func ParseSession(header string) string {
	mparsed := strings.Split(header, "\r\n")
	for _, element := range mparsed {
		if strings.Contains(element, "Session:") {
			if strings.Contains(element, ";") {
				fist := strings.Split(element, ";")[0]
				return fist[9:]
			} else {
				return element[9:]
			}
		}
	}
	return ""
}
func ParseMedia(header string) []string {
	letters := []string{}
	mparsed := strings.Split(header, "\r\n")
	paste := ""

	if true {
		log.Println("headers", header)
	}

	for _, element := range mparsed {
		if strings.Contains(element, "a=control:") && !strings.Contains(element, "*") && strings.Contains(element, "tra") {
			paste = element[10:]
			if strings.Contains(element, "/") {
				striped := strings.Split(element, "/")
				paste = striped[len(striped)-1]
			}
			letters = append(letters, paste)
		}

		dimensionsPrefix := "a=x-dimensions:"
		if strings.HasPrefix(element, dimensionsPrefix) {
			dims := []int{}
			for _, s := range strings.Split(element[len(dimensionsPrefix):], ",") {
				v := 0
				fmt.Sscanf(s, "%d", &v)
				if v <= 0 {
					break
				}
				dims = append(dims, v)
			}
			if len(dims) == 2 {
				VideoWidth = dims[0]
				VideoHeight = dims[1]
			}
		}
	}
	return letters
}
func GetMD5Hash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}
func (this *RtspClient) ParseMedia(header string) []string {
	letters := []string{}
	mparsed := strings.Split(header, "\r\n")
	paste := ""
	for _, element := range mparsed {
		if strings.Contains(element, "a=control:") && !strings.Contains(element, "*") && strings.Contains(element, "tra") {
			paste = element[10:]
			if strings.Contains(element, "/") {
				striped := strings.Split(element, "/")
				paste = striped[len(striped)-1]
			}
			letters = append(letters, paste)
		}

		dimensionsPrefix := "a=x-dimensions:"
		if strings.HasPrefix(element, dimensionsPrefix) {
			dims := []int{}
			for _, s := range strings.Split(element[len(dimensionsPrefix):], ",") {
				v := 0
				fmt.Sscanf(s, "%d", &v)
				if v <= 0 {
					break
				}
				dims = append(dims, v)
			}
			if len(dims) == 2 {
				this.videow = dims[0]
				this.videoh = dims[1]
			}
		}
	}
	return letters
}
