package lock

import (
	"errors"
	"sync"

	"time"

	"fmt"

	"net/http"

	"io/ioutil"

	"net/url"

	"github.com/Sirupsen/logrus"
	"github.com/json-iterator/go"
	"github.com/patrickmn/go-cache"
	"github.com/prometheus/common/log"
	"github.com/robfig/cron"
)

type IP struct {
	IP    string
	Port  string
	mu    sync.RWMutex
	count int
}

func (ip *IP) Use() {
	ip.mu.Lock()
	defer ip.mu.Unlock()
	ip.count++
}

func (ip *IP) Check() bool {
	ip.mu.Lock()
	defer ip.mu.Unlock()
	return ip.count < 3
}

func (ip *IP) CheckAndUse() bool {

	logrus.Infof("Check IP: %s", ip.IP)
	ip.mu.Lock()
	defer ip.mu.Unlock()
	if ip.count < 5 {
		ip.count++
		return true
	} else {
		return false
	}
}

type IPPool struct {
	Min      int
	Max      int
	IPCount  int
	ApiAddr  string
	Interval string

	cron  *cron.Cron
	cache *cache.Cache
}

func (pool *IPPool) PutIP() {

	if len(pool.cache.Items()) >= pool.Max {
		return
	}

	resp, err := http.Get(pool.ApiAddr)
	if err != nil {
		logrus.Errorln(err)
		return
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.Errorln(err)
		return
	}

	// clean expired ip
	pool.cache.DeleteExpired()

	var ips []IP
	jsoniter.UnmarshalFromString(jsoniter.Get(b, "msg").ToString(), &ips)
	if err != nil {
		logrus.Errorln(err)
		return
	}
	for _, ip := range ips {
		logrus.Infof("Pool put ip: %s", ip.IP)

		proxyAddr := fmt.Sprintf("http://%s:%s", ip.IP, ip.Port)

		p := func(_ *http.Request) (*url.URL, error) {
			return url.Parse(proxyAddr)
		}
		transport := &http.Transport{Proxy: p}

		client := &http.Client{
			Timeout:   3 * time.Second,
			Transport: transport,
		}
		req, _ := http.NewRequest("GET", "https://www.apple.com", nil)
		_, err := client.Do(req)
		if err != nil {
			log.Warnf("IP [%s] unavailable, skip!", ip.IP)
		} else {
			pool.cache.Set(ip.IP, ip, 30*time.Second)
		}
	}

}

func (pool *IPPool) GetIP() (*IP, error) {
	items := pool.cache.Items()
	for _, it := range items {
		ip := it.Object.(IP)
		if ip.CheckAndUse() {
			return &ip, nil
		} else {
			pool.cache.Delete(ip.IP)
		}
	}

	return nil, errors.New("no ip available")
}

func (pool *IPPool) WaitReady() chan int {
	readyCh := make(chan int, 1)

	go func() {
		for range time.Tick(1 * time.Second) {
			if pool.cache.ItemCount() >= pool.Min {
				logrus.Infoln("IP Pool is ready!")
				readyCh <- 1
			} else {
				logrus.Infoln("IP Pool not ready!")
			}
		}
	}()
	return readyCh
}

func (pool *IPPool) Start() error {

	if pool.Min == 0 {
		pool.Min = 20
	}
	if pool.Max == 0 {
		pool.Max = 30
	}
	if pool.Interval == "" {
		pool.Interval = "10s"
	}
	if pool.cache == nil {
		pool.cache = cache.New(3*time.Minute, 5*time.Minute)
	}

	_, err := time.ParseDuration(pool.Interval)
	if err != nil {
		return err
	}

	pool.cron = cron.New()
	pool.cron.AddFunc(fmt.Sprintf("@every %s", pool.Interval), func() {
		logrus.Infoln("IP Pool cron running...")
		pool.PutIP()
	})
	logrus.Infoln("IP Pool starting...")
	pool.cron.Start()

	return nil
}
