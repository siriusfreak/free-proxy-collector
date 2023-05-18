package free_proxy_cz

import (
	"context"
	"net"
	"strconv"
	"strings"

	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"

	"github.com/siriusfreak/free-proxy-gun/collection"
	"github.com/siriusfreak/free-proxy-gun/log"
)

const (
	collectorName = "free-proxy-cz"
	baseUri       = "https://www.freeproxylists.net/"

	xPath = "/html/body/div[1]/div[2]/table/tbody/tr"
)

type Collector struct {
	reCaptchaByPasser ReCaptchaByPasser
}

type ReCaptchaByPasser interface {
	WaitForResult(ctx context.Context, requestID string) (string, error)
	AddRequest(ctx context.Context, pageUrl, dataSiteKey string) (requestID string, err error)
}

var validTags = map[string]struct{}{
	"Odd":  {},
	"Even": {},
}

func New(reCaptchaByPasser ReCaptchaByPasser) *Collector {
	return &Collector{
		reCaptchaByPasser: reCaptchaByPasser,
	}
}
func (c *Collector) GetName() string {
	return collectorName
}

func parseLine(ctx context.Context, tr *html.Node) (collection.Proxy, error) {
	res := collection.Proxy{}
	curChild := tr.FirstChild
	childInd := 0
	var err error
	for curChild != nil {
		txt := htmlquery.InnerText(curChild)
		switch childInd {
		case 0:
			txt = txt[strings.Index(txt, ")")+1:]
			res.IP = net.ParseIP(txt)
		case 1:
			res.Port, err = strconv.Atoi(txt)
			if err != nil {
				log.Warn(ctx, "cannot convert port")
				continue
			}
		case 2:
			res.Type = collection.ParseProxyType(txt)
		case 3:
			res.AnonymousLevel = collection.ParseAnonymousLevel(txt)
		case 4:
			res.Country = strings.Trim(txt, " ")
		}

		childInd++
		curChild = curChild.NextSibling
	}

	return res, nil
}

func (c *Collector) parseTable(ctx context.Context, page string) ([]collection.Proxy, error) {
	res := make([]collection.Proxy, 0)

	doc, err := htmlquery.Parse(strings.NewReader(page))
	if err != nil {
		return nil, err
	}

	trs := htmlquery.Find(doc, xPath)
	for _, tr := range trs {
		goodLine := false
		for _, attr := range tr.Attr {
			if _, ok := validTags[attr.Val]; ok {
				goodLine = true
				break
			}
		}

		if goodLine {
			cur, err := parseLine(ctx, tr)
			if err != nil {
				log.Warn(ctx, "Parse error")
			} else {
				res = append(res, cur)
			}

		}
	}

	return res, err
}

func (c *Collector) Collect(ctx context.Context) ([]collection.Proxy, error) {
	res := make([]collection.Proxy, 0)
	nav := newNavigator(ctx)

	err := nav.loadFirstPage(baseUri)
	if err != nil {
		return nil, err
	}

	for {
		isCaptcha, err := nav.isCaptcha()
		if err != nil {
			return nil, err
		}

		if isCaptcha {
			log.Info(ctx, "captcha found")
			err := nav.bypassCaptcha(ctx, c.reCaptchaByPasser)
			if err != nil {
				return nil, err
			}
		} else {
			log.Info(ctx, "captcha not found")
		}

		page, err := nav.getPage()
		if err != nil {
			return nil, err
		}

		log.Info(ctx, "start parsing")
		list, err := c.parseTable(ctx, page)
		if err != nil {
			return nil, err
		}
		res = append(res, list...)

		if nav.loadNextPage() != nil {
			break
		}
	}

	return res, nil
}
