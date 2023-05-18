package free_proxy_cz

import (
	"context"
	"errors"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
)

const (
	reCaptchaXPath           = "//*[@id=\"contents\"]/center/form/div"
	dataSitekeyAttributeName = "data-sitekey"
)

type navigator struct {
	taskCtx context.Context

	cancellations []func()
}

var (
	errNoReCapthaDataSitekeyFound = errors.New("No ReCaptcha data-sitekey found")
)

func newNavigator(ctx context.Context) *navigator {
	options := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag(`headless`, false),
		chromedp.DisableGPU,
		chromedp.Flag(`disable-extensions`, false),
		chromedp.Flag(`enable-automation`, false),
		chromedp.UserAgent(`Mozilla/5.0 (Windows NT 6.3; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/73.0.3683.103 Safari/537.36`),
	)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, options...)
	taskCtx, cancelCtx := chromedp.NewContext(allocCtx)

	return &navigator{
		taskCtx:       taskCtx,
		cancellations: []func(){cancelAlloc, cancelCtx},
	}
}

func (n *navigator) loadFirstPage(url string) error {
	return chromedp.Run(n.taskCtx,
		chromedp.Navigate(url),
		chromedp.WaitReady("body"),
	)

}

func (n *navigator) loadNextPage() error {
	return chromedp.Run(n.taskCtx,
		chromedp.Click(`body > div:nth-child(3) > div:nth-child(2) > div:nth-child(7) > a:nth-child(3)`),
		chromedp.WaitReady("body"),
	)
}

func (n *navigator) getPage() (string, error) {
	html := ""
	err := chromedp.Run(n.taskCtx,
		chromedp.OuterHTML("html", &html),
	)
	if err != nil {
		return "", err
	}
	return html, nil
}

func (n *navigator) isCaptcha() (bool, error) {
	nodes := make([]*cdp.Node, 0)
	err := chromedp.Run(n.taskCtx,
		chromedp.Nodes("#contents > center > form > input[type=submit]", &nodes, chromedp.AtLeast(0)),
	)
	if err != nil {
		return false, err
	}
	if len(nodes) == 0 {
		return false, nil
	}
	return true, nil
}

func (n *navigator) bypassCaptcha(ctx context.Context, reCaptchaByPasser ReCaptchaByPasser) error {
	var nodes []*cdp.Node
	var curUrl string
	err := chromedp.Run(n.taskCtx,
		chromedp.Nodes(reCaptchaXPath, &nodes),
		chromedp.Location(&curUrl),
	)
	if err != nil {
		return err
	}

	if len(nodes) == 0 {
		return errNoReCapthaDataSitekeyFound
	}

	dataSiteKey, ok := nodes[0].Attribute(dataSitekeyAttributeName)
	if !ok {
		return errNoReCapthaDataSitekeyFound
	}

	requestID, err := reCaptchaByPasser.AddRequest(ctx, curUrl, dataSiteKey)
	if err != nil {
		return err
	}
	googleKey, err := reCaptchaByPasser.WaitForResult(ctx, requestID)
	if err != nil {
		return err
	}

	err = chromedp.Run(n.taskCtx,
		chromedp.EvaluateAsDevTools("document.getElementById(\"g-recaptcha-response\").innerHTML=\""+googleKey+"\";", nil),
		chromedp.EvaluateAsDevTools("document.querySelector(\"#contents > center > form\").submit()", nil),
	)

	if err != nil {
		return err
	}

	return nil
}
