package ru_captcha

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/siriusfreak/free-proxy-gun/log"
	"go.uber.org/zap"
	"io"
	"net/http"
	url "net/url"
	"sync"
	"time"
)

const (
	ruCaptchaInURL  = "http://rucaptcha.com/in.php"
	ruCaptchaResURL = "http://rucaptcha.com/res.php"

	methodPOST = "POST"
	methodGET  = "GET"
)

type ruCaptchaResponse struct {
	Status  int    `json:"status"`
	Request string `json:"request"`
}

type RateLimiter interface {
	Acquire(ctx context.Context) error
}

type RuCaptcha struct {
	applicationContext context.Context
	apiKey             string
	timeOut            time.Duration
	rateLimiter        RateLimiter

	mu       *sync.Mutex
	waitList map[string]chan string
}

var (
	errCapthaRequestFailed = errors.New("captcha request failed")
	errStatusCodeNotOk     = errors.New("status code not OK")
	errCapthaNotReady      = errors.New("captcha not ready")
)

func New(ctx context.Context, apiKey string, rateLimiter RateLimiter, timeOut time.Duration) *RuCaptcha {
	rc := &RuCaptcha{
		applicationContext: ctx,
		apiKey:             apiKey,
		timeOut:            timeOut,
		rateLimiter:        rateLimiter,
		mu:                 &sync.Mutex{},
		waitList:           make(map[string]chan string),
	}
	go rc.resultFetcher(ctx)

	return rc
}

func (rc *RuCaptcha) requestResult(ctx context.Context, requestID string) (string, error) {
	err := rc.rateLimiter.Acquire(ctx)
	if err != nil {
		return "", err
	}

	params := url.Values{}
	params.Add("key", "6717723836740ce75cfb01766c73c3df")
	params.Add("action", "get")
	params.Add("id", requestID)
	params.Add("json", "1")

	baseURL, err := url.Parse(ruCaptchaResURL)
	if err != nil {
		return "", err
	}
	baseURL.RawQuery = params.Encode()

	req, err := http.NewRequest(methodGET, baseURL.String(), bytes.NewBufferString(params.Encode()))
	if err != nil {
		return "", err
	}
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", errStatusCodeNotOk
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	rr := ruCaptchaResponse{}
	err = json.Unmarshal(body, &rr)
	if err != nil {
		return "", err
	}
	if rr.Status == 0 {
		return "", errCapthaNotReady
	}

	return rr.Request, nil

}
func (rc *RuCaptcha) resultFetcher(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		for k, v := range rc.waitList {
			requestResult, err := rc.requestResult(ctx, k)
			if err != nil {
				log.Info(ctx, "requestResult not OK", zap.Error(err), zap.String("requestID", k))
				continue
			}
			v <- requestResult
			rc.mu.Lock()
			close(v)
			delete(rc.waitList, k)
			rc.mu.Unlock()
		}

		// To avoid active waiting
		time.Sleep(10 * time.Millisecond)
	}
}

func (rc *RuCaptcha) WaitForResult(ctx context.Context, requestID string) (string, error) {
	select {
	case res := <-rc.waitList[requestID]:
		return res, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (rc *RuCaptcha) AddRequest(ctx context.Context, pageUrl, dataSiteKey string) (requestID string, err error) {
	postBody, err := json.Marshal(map[string]string{
		"key":       "6717723836740ce75cfb01766c73c3df",
		"method":    "userrecaptcha",
		"googlekey": dataSiteKey,
		"pageurl":   pageUrl,
		"invisible": "1",
		"json":      "1",
	})
	if err != nil {
		return "", err
	}

	client := &http.Client{
		Timeout: rc.timeOut,
	}

	req, err := http.NewRequest(methodPOST, ruCaptchaInURL, bytes.NewReader(postBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	jobInfo := ruCaptchaResponse{}
	err = json.Unmarshal(body, &jobInfo)
	if err != nil {
		return "", err
	}

	if jobInfo.Status != 1 {
		log.Warn(ctx, "ru-captcha: request failed", "status", jobInfo.Status, "request", jobInfo.Request)
		return "", errCapthaRequestFailed
	}

	rc.mu.Lock()
	rc.waitList[jobInfo.Request] = make(chan string)

	return jobInfo.Request, nil
}
