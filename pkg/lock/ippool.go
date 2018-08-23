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

const apiTpl = "https://proxy.horocn.com/api/proxies?order_id=%s&num=%d&format=json&line_separator=unix"

type IP struct {
	Host        string
	Port        string
	Country_cn  string
	Province_cn string
	City_cn     string

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
	ip.mu.Lock()
	defer ip.mu.Unlock()
	if ip.count < 3 {
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
	OrderID  string
	Interval string

	cron  *cron.Cron
	cache *cache.Cache
}

func (pool *IPPool) PutIP() {

	if pool.cache.ItemCount() >= pool.Max {
		return
	}

	apiAddr := fmt.Sprintf(apiTpl, pool.OrderID, pool.IPCount)
	resp, err := http.Get(apiAddr)
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

	var ips []IP
	err = jsoniter.Unmarshal(b, &ips)
	if err != nil {
		logrus.Errorln(err)
		return
	}
	for _, ip := range ips {
		logrus.Infof("Pool put ip: %s", ip.Host)

		proxyAddr := fmt.Sprintf("http://%s:%s", ip.Host, ip.Port)

		p := func(_ *http.Request) (*url.URL, error) {
			return url.Parse(proxyAddr)
		}
		transport := &http.Transport{Proxy: p}

		client := &http.Client{
			Timeout:   5 * time.Second,
			Transport: transport,
		}
		req, _ := http.NewRequest("GET", "https://www.baidu.com", nil)
		_, err := client.Do(req)
		if err != nil {
			log.Warnf("IP [%s] unavailable, skip!", ip.Host)
		} else {
			pool.cache.Set(ip.Host, ip, 3*time.Minute)
		}
	}

}

func (pool *IPPool) GetIP() (*IP, error) {
	items := pool.cache.Items()
	for _, it := range items {
		ip := it.Object.(IP)
		if ip.CheckAndUse() {
			return &ip, nil
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
	if pool.OrderID == "" {
		return errors.New("ip pool order id is blank")
	}
	if pool.Min == 0 {
		pool.Min = 20
	}
	if pool.Max == 0 {
		pool.Max = 30
	}
	if pool.IPCount == 0 {
		pool.IPCount = 10
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
