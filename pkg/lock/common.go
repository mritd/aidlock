package lock

import (
	"math/rand"
	"net/http"
	"os"
	"os/user"

	"github.com/Sirupsen/logrus"
)

const (
	UA             = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/68.0.3440.84 Safari/537.36"
	Host           = "idmsa.apple.com"
	WidgetKey      = "af1139274f266b22b68c2a3e7ad932cb3c0bbe854e13a79af78dcc73136882c3"
	BaseURL        = "https://idmsa.apple.com"
	Referer        = BaseURL + "/appleauth/auth/signin?widgetKey=" + WidgetKey + "&language=zh_CN&rv=1"
	AcceptEncoding = ""
	AcceptLanguage = "zh-CN,zh;q=0.9,en;q=0.8"
	AcceptJSON     = "application/json, text/javascript, */*; q=0.01"
	AcceptHTML     = "text/html;format=fragmented"
	ContentType    = "application/json"
)

const (
	letterBytes   = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

func RandString(n int) string {
	b := make([]byte, n)
	// A rand.Int63() generates 63 random bits, enough for letterIdxMax letters!
	for i, cache, remain := n-1, rand.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = rand.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

func CheckErr(err error) bool {
	if err != nil {
		logrus.Infoln(err)
		return false
	}
	return true
}

func CheckAndExit(err error) {
	if !CheckErr(err) {
		os.Exit(1)
	}
}

func CheckRoot() {
	u, err := user.Current()
	CheckAndExit(err)

	if u.Uid != "0" || u.Gid != "0" {
		logrus.Infoln("This command must be run as root! (sudo)")
		os.Exit(1)
	}
}

func setCommonHeader(req *http.Request) {

	req.Header.Set("User-Agent", UA)
	req.Header.Set("Accept-Encoding", AcceptEncoding)
	req.Header.Set("Accept-Language", AcceptLanguage)
	req.Header.Set("Content-Type", ContentType)
	req.Header.Set("Host", Host)
	req.Header.Set("Origin", BaseURL)
	req.Header.Set("Referer", Referer)
	req.Header.Set("Accept", AcceptJSON)

}
