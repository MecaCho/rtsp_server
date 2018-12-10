package rtspclient

import (
	"crypto/md5"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"time"
)

func DecKeytool(value string) (string, error) {
	return "", nil
}

const (
	USRAGENT = "LibVLC/3.0.0 (LIVE555 Streaming Media v2016.11.28)"
)

//RTSPClient ...
type RTSPClient struct {
	address  string
	url      string
	host     string
	port     string
	conn     net.Conn
	CHECKLOG string
}

func genmsgDESCRIBE(url, seq, userAgent string) string {
	msgRet := "DESCRIBE " + url + " RTSP/1.0\r\n"
	msgRet += "CSeq: " + seq + "\r\n"
	msgRet += "User-Agent: " + userAgent + "\r\n"
	msgRet += "Accept: application/sdp\r\n"
	msgRet += "\r\n"
	return msgRet
}

func convMd5(obj string) string {
	md5 := md5.New()
	io.WriteString(md5, obj)
	return fmt.Sprintf("%x", md5.Sum(nil))
}

//buildRequest ...
func buildRequest(method, realm, nonce, url, pwd, usr string) string {
	opMd5 := convMd5(fmt.Sprintf("%s:%s", method, url))
	pwdMd5 := convMd5(fmt.Sprintf("%s:%s:%s", usr, realm, pwd))
	obj := pwdMd5 + ":" + nonce + ":" + opMd5
	return convMd5(obj)
}

//doRequest ...
func (rtspClient *RTSPClient) doRequest(address, options string) (string, string, error) {
	conn := rtspClient.conn
	num, err := conn.Write([]byte(options))
	if err != nil {
		return "", "", fmt.Errorf("Connect Write Error : %q , Num : %q", err, num)
	}

	buffer := make([]byte, 4096*2)
	nb, err := conn.Read(buffer)
	if err != nil || nb <= 0 {
		return "", "", fmt.Errorf("Net connnection read failed : %q", err)
	}
	readString := string(buffer[:nb])
	statusCode := strings.TrimSpace(strings.Split(readString, "\n")[0])
	return readString, statusCode, nil
}

//GetRealmNonce ...
func (rtspClient *RTSPClient) GetRealmNonce(address, url, usrAgent string, seq int) (string, string, string, error) {
	op := genmsgDESCRIBE(url, "1", USRAGENT)
	ret, _, err := rtspClient.doRequest(address, op)
	if err != nil {
		return "", "", "", fmt.Errorf("Send message Error:%q", err)
	}
	lines := strings.Split(ret, "\n")
	respCode := strings.TrimSpace(lines[0])
	if !strings.Contains(lines[0], "RTSP/1.0 401 Unauthorized") {
		return "", "", respCode, fmt.Errorf("Get realm and nonce Error: rtsp responce not 401")
	}
	nonce := strings.Split(strings.Split(lines[2], ",")[1], "\"")[1]
	realm := strings.Split(strings.Split(lines[2], ",")[0], "\"")[1]
	return realm, nonce, respCode, nil
}

func genMsg(method, url, usr, nonce, responce string) string {
	ret := fmt.Sprintf("%s %s RTSP/1.0\r\n", method, url)
	ret += "CSeq: 3\r\n"
	ret += fmt.Sprintf("Authorization: Digest username=\"%s\", realm=\"Embedded Net DVR\", nonce=\"%s\", uri=\"%s\", response=\"%s\"\r\n", usr, nonce, url, responce)
	ret += "User-Agent: LibVLC/3.0.0 (LIVE555 Streaming Media v2016.11.28)\r\n\r\n"
	return ret
}

//CheckMain ...
func CheckMain(rtsp_url string) (string, error) {
	deviceRealURL := rtsp_url
	if len(strings.Split(deviceRealURL, "@")) != 2 {
		logURL := deviceRealURL
		rtspClient, err := NewRTSPClient(deviceRealURL)
		rtspClient.CHECKLOG = fmt.Sprintf("Connect to (%s) , send message to (%s) :", rtspClient.address, logURL)
		if err != nil {
			return rtspClient.CHECKLOG, fmt.Errorf("Connect rtsp error :%s", err.Error())
		}
		_, _, respCode, err := rtspClient.GetRealmNonce(rtspClient.address, deviceRealURL, USRAGENT, 1)
		rtspClient.CHECKLOG += fmt.Sprintf(" DESCRIBE --> %s , ", respCode)
		if err != nil {
			if strings.Contains(respCode, "200") {
				return rtspClient.CHECKLOG, nil
			}
		}
		return rtspClient.CHECKLOG, fmt.Errorf("Camera url format error : has no password and usr in url")
	}

	logURL := "rtsp://" + strings.Split(deviceRealURL, "@")[1]
	Credentials := strings.Split(strings.Split(deviceRealURL, "@")[0], "//")[1]
	pwd := strings.Split(Credentials, ":")[1]
	usr := strings.Split(Credentials, ":")[0]

	rtspClient, err := NewRTSPClient(deviceRealURL)
	rtspClient.CHECKLOG = fmt.Sprintf("Connect to (%s) , send message to (%s) :", rtspClient.address, logURL)
	if err != nil {
		return rtspClient.CHECKLOG, fmt.Errorf("Connect rtsp error :%s", err.Error())
	}

	realm, nonce, respCode, err := rtspClient.GetRealmNonce(rtspClient.address, deviceRealURL, USRAGENT, 1)
	rtspClient.CHECKLOG += fmt.Sprintf(" DESCRIBE --> %s , ", respCode)
	if err != nil {
		if strings.Contains(respCode, "200") {
			return rtspClient.CHECKLOG, nil
		}
		return rtspClient.CHECKLOG, fmt.Errorf("GetRealmNonce Error : %s", err.Error())
	}

	resp := buildRequest("OPTIONS", realm, nonce, deviceRealURL, pwd, usr)
	options := genMsg("OPTIONS", logURL, usr, nonce, resp)
	_, responce, err := rtspClient.doRequest(rtspClient.address, options)
	rtspClient.CHECKLOG += fmt.Sprintf(" OPTIONS --> %s . ", responce)
	if err != nil {
		return rtspClient.CHECKLOG, fmt.Errorf("doRequest error : %q , Reponce : (%q)", err, responce)
	}

	desResp := buildRequest("DESCRIBE", realm, nonce, logURL, pwd, usr)
	des := genMsg("DESCRIBE", logURL, usr, nonce, desResp)
	_, ret, err := rtspClient.doRequest(rtspClient.address, des)
	rtspClient.CHECKLOG += fmt.Sprintf(" DESCRIBE --> %s . ", ret)
	if strings.Contains(ret, "200 OK") {
		return rtspClient.CHECKLOG, nil
	}
	return rtspClient.CHECKLOG, fmt.Errorf("doRequest error : %q , Reponce : (%q)", err, ret)
}

func (this *RTSPClient) ParseCameraUrl(rtsp_url string) {
	this.url = rtsp_url
	u, err := url.Parse(rtsp_url)
	if err != nil {
		fmt.Println(err.Error())
	}
	this.address = u.Host
	phost := strings.Split(this.address, ":")
	this.host = phost[0]
	if len(phost) == 2 {
		this.port = phost[1]
	} else {
		this.port = "554"
	}
}

//NewRTSPClient ...
func NewRTSPClient(url string) (*RTSPClient, error) {
	rtsp := new(RTSPClient)
	rtsp.ParseCameraUrl(url)
	rtspcon, err := net.DialTimeout("tcp", rtsp.address, 5*time.Second)
	if err != nil {
		return rtsp, fmt.Errorf("connect to remote camera err : %s", err.Error())
	}
	rtsp.conn = rtspcon
	return rtsp, nil
}

func main() {
	u, err := url.Parse("rtsp://admin:123456@100.114.234.225:554/mpeg4cif.sdp")
	//&url.URL{Scheme:"rtsp", Opaque:"", User:(*url.Userinfo)(0xc42007a510), Host:"100.114.234.225:554", Path:"/mpeg4cif.sdp", RawPath:"", ForceQuery:false, RawQuery:"", Fragment:""}
	//Process finished with exit code 0
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Printf("%#v", u)
	//log, err := CheckMain("rtsp://admin:123456@100.114.234.225:554/mpeg4cif.sdp")
	//if err != nil{
	//	fmt.Printf(err.Error())
	//}
	//fmt.Println(log)
}
