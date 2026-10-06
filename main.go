package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var (
	DomainSuffix string
	Bind         string
	ProxyUrl     string
	Timeout      time.Duration
)

func init() {
	flag.StringVar(&Bind, "bind", ":8000", "")
	flag.StringVar(&ProxyUrl, "proxy", "", "")
	flag.DurationVar(&Timeout, "timeout", time.Second*5, "")
	flag.Parse()
	if flag.NArg() < 1 {
		fmt.Println("Usage: exec DOMAIN_SUFFIX")
		os.Exit(1)
	}
	DomainSuffix = flag.Arg(0)
}

func main() {
	proxy := http.ProxyFromEnvironment
	if ProxyUrl != "" {
		proxyUrl, err := url.Parse(ProxyUrl)
		if err != nil {
			log.Fatal(err)
		}
		proxy = http.ProxyURL(proxyUrl)
	}
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: &http.Transport{
			Proxy: proxy,
		},
		Timeout: Timeout,
	}
	if err := http.ListenAndServe(Bind, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.URL.Scheme = r.Header.Get("X-Forwarded-Proto")
		if r.URL.Scheme == "" {
			r.URL.Scheme = "https"
		}
		r.RequestURI = ""
		r.Host = strings.TrimSuffix(r.Host, "."+DomainSuffix)
		r.URL.Host = r.Host
		r.Header.Set("Host", r.Host)
		for k, _ := range r.Header {
			if strings.HasPrefix(k, "X-Forwarded-") {
				r.Header.Del(k)
			}
		}
		resp, err := client.Do(r)
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()
		h := w.Header()
		for k, vv := range resp.Header {
			for _, v := range vv {
				if k == "Location" {
					u, err := url.Parse(v)
					if err != nil {
						fmt.Println(err)
						h.Add(k, v)
					} else {
						u.Host += "." + DomainSuffix
						h.Add(k, u.String())
					}
				} else {
					h.Add(k, v)
				}
			}
		}
		w.WriteHeader(resp.StatusCode)
		if _, err := io.Copy(w, resp.Body); err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})); err != nil {
		panic(err)
	}
}
