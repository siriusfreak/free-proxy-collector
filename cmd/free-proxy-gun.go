package main

import (
	"context"
	"github.com/siriusfreak/free-proxy-gun/collection"
	freeProxyCz "github.com/siriusfreak/free-proxy-gun/collection/collectors/free-proxy-cz"
	ru_captcha "github.com/siriusfreak/free-proxy-gun/helpers/captcha/ru-captcha"
	rate_limiter "github.com/siriusfreak/free-proxy-gun/helpers/rate-limiter"
	"github.com/siriusfreak/free-proxy-gun/log"
	"go.uber.org/zap"
	"time"
)

func main() {
	ctx := context.Background()

	rateLimiter := rate_limiter.New(ctx, 10)
	reCapchaByPasser := ru_captcha.New(ctx, "6717723836740ce75cfb01766c73c3df", rateLimiter, time.Second*10)
	collectors := []collection.Collector{freeProxyCz.New(reCapchaByPasser)}
	res, err := collection.Collect(ctx, collectors)
	log.Info(ctx, "collected", zap.Int("count", len(res)), zap.Error(err))
}
