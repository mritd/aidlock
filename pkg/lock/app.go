package lock

import (
	"os"

	"net/http"
	"net/http/cookiejar"
	"time"

	"net/url"

	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/robfig/cron"
	"github.com/spf13/viper"
)

func Boot() {
	var appleIDs []AppleID
	err := viper.UnmarshalKey("AppleIDs", &appleIDs)
	if err != nil {
		logrus.Infoln("Can't parse Apple ID config!")
		os.Exit(1)
	}

	var cronStr string
	err = viper.UnmarshalKey("cron", &cronStr)
	if err != nil {
		logrus.Infoln("Can't parse cron config!")
		os.Exit(1)
	}

	var pool IPPool
	err = viper.UnmarshalKey("pool", &pool)
	if err != nil {
		logrus.Infoln("Can't parse ip pool config!")
		os.Exit(1)
	}
	err = pool.Start()
	if err != nil {
		logrus.Infof("Start IP Pool failed: %s", err)
		os.Exit(1)
	}
	logrus.Infoln("IP Pool started, wait pool ready...")
	<-pool.WaitReady()

	c := cron.New()
	for i := range appleIDs {
		x := i

		logrus.Infof("Apple ID [%s] cron starting", appleIDs[x].ID)

		c.AddFunc(cronStr, func() {

			for i := 0; i < 20; i++ {

				jar, _ := cookiejar.New(nil)

				ip, err := pool.GetIP()
				if err != nil {
					logrus.Errorf("Apple lock failed: %s", err)
					return
				}
				logrus.Infof("Lock apple id [%s] use IP: %s", appleIDs[x].ID, ip.Host)
				proxyAddr := fmt.Sprintf("http://%s:%s", ip.Host, ip.Port)

				p := func(_ *http.Request) (*url.URL, error) {
					return url.Parse(proxyAddr)
				}
				transport := &http.Transport{Proxy: p}

				client := &http.Client{
					Timeout:   5 * time.Second,
					Jar:       jar,
					Transport: transport,
				}
				if appleIDs[x].Lock(client) {
					break
				}
			}
			if appleIDs[x].State {
				logrus.Infof("Apple ID [%s] locked!", appleIDs[x].ID)
			}
		})
	}
	c.Start()
	select {}
}
