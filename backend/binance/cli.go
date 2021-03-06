package binance

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Client struct {
	h         *http.Client
	base      string
	apiKey    string
	secretKey []byte
}

func NewBinance(h *http.Client, base string, apiKey string, secretKey []byte) *Client {
	return &Client{h: h, base: base, apiKey: apiKey, secretKey: secretKey}
}

const (
	apiKeyHeader = "X-MBX-APIKEY"

	pingPath                = "api/v3/ping"
	exchangeInfoPath        = "api/v3/exchangeInfo"
	myTradesPath            = "api/v3/myTrades"
	accountPath             = "api/v3/account"
	allOrdersPath           = "api/v3/allOrders"
	currentAveragePricePath = "api/v3/avgPrice"
	timeCheckPath           = "api/v3/time"
	orderPath               = "api/v3/order"
)

func (c Client) Ping() {
	url := fmt.Sprintf("%s/%s", c.base, pingPath)
	fmt.Println(c.h.Get(url))
}

func (c Client) ExchangeInfo() {
	url := fmt.Sprintf("%s/%s", c.base, exchangeInfoPath)
	r, _ := c.h.Get(url)
	body, _ := ioutil.ReadAll(r.Body)
	fmt.Println(string(body))
}

/*
https://githuc.com/binance/binance-spot-api-docs/blob/master/rest-api.md#account-trade-list-user_data
NAME		TYPE 	MANDATORY 	DESCRIPION
===========================================
symbol		STRING	YES
startTime	LONG	NO
endTime		LONG	NO
fromId		LONG	NO			TradeId to fetch from. Default gets most recent trades.
limit		INT		NO			Default 500; max 1000.
recvWindow	LONG	NO			The value cannot be greater than 60000
timestamp	LONG	YES
*/

// MyTrades returns list of completed trades
func (c Client) MyTrades(req MyTradesRequest) (*MyTradesResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("%s/%s", c.base, myTradesPath)
	parsedURL, err := url.Parse(u)
	if err != nil {
		return nil, err
	}

	r, _ := http.NewRequest(http.MethodGet, c.createURL(req, parsedURL), nil)
	r.Header.Set(apiKeyHeader, c.apiKey)

	var trades MyTradesResponse

	resp, err := c.h.Do(r)
	if err != nil {
		return nil, err
	}
	if err := json.NewDecoder(resp.Body).Decode(&trades.Trades); err != nil {
		return nil, err
	}

	return &trades, nil
}

func (c Client) Account(req AccountRequest) (*AccountResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("%s/%s", c.base, accountPath)
	parsedURL, err := url.Parse(u)
	if err != nil {
		return nil, err
	}

	r, _ := http.NewRequest(http.MethodGet, c.createURL(req, parsedURL), nil)
	r.Header.Set(apiKeyHeader, c.apiKey)

	fmt.Println(c.createURL(req, parsedURL))

	var trades AccountResponse

	resp, err := c.h.Do(r)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, parseError(body)
	}
	if err := json.Unmarshal(body, &trades); err != nil {
		return nil, err
	}
	return &trades, nil
}

func (c Client) GetServerTime() (time.Time, error) {
	u := fmt.Sprintf("%s/%s", c.base, timeCheckPath)
	parsedURL, err := url.Parse(u)
	if err != nil {
		return time.Time{}, err
	}
	r, _ := http.NewRequest(http.MethodGet, parsedURL.String(), nil)
	resp, err := c.h.Do(r)
	if err != nil {
		return time.Time{}, err
	}

	var str ServerTimeResponse
	if err := json.NewDecoder(resp.Body).Decode(&str); err != nil {
		return time.Time{}, err
	}
	return time.Unix(str.ServerTime/1000., 0), nil
}

func (c Client) AllOrderList(req AllOrdersRequest) (*AllOrdersResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("%s/%s", c.base, allOrdersPath)
	parsedURL, err := url.Parse(u)
	if err != nil {
		return nil, err
	}

	r, _ := http.NewRequest(http.MethodGet, c.createURL(req, parsedURL), nil)
	r.Header.Set(apiKeyHeader, c.apiKey)

	var orders AllOrdersResponse

	resp, err := c.h.Do(r)
	if err != nil {
		return nil, err
	}
	if err := json.NewDecoder(resp.Body).Decode(&orders.Orders); err != nil {
		return nil, err
	}

	return &orders, nil
}

// TODO fix creating url + reading error

func (c Client) CurrentAveragePrice(symbol string) (float64, error) {
	u := fmt.Sprintf("%s/%s", c.base, currentAveragePricePath)

	parsedURL, err := url.Parse(u)
	if err != nil {
		return 0, err
	}
	r, _ := http.NewRequest(http.MethodGet, parsedURL.String()+fmt.Sprintf("?symbol=%s", symbol), nil)
	r.Header.Set(apiKeyHeader, c.apiKey)

	resp, err := c.h.Do(r)
	if err != nil {
		return 0, err
	}
	var capResp CurrentAveragePriceResponse
	if err := json.NewDecoder(resp.Body).Decode(&capResp); err != nil {
		return 0, err
	}
	//if resp.StatusCode != http.StatusOK {
	//	return 0, parseError(resp.Body)
	//}

	fl, err := strconv.ParseFloat(capResp.Price, 64)
	if err != nil {
		return 0, err
	}
	return fl, nil
}

func (c Client) SymbolTickerPrice(symbol string) (float64, error) {
	u := fmt.Sprintf("%s/%s", c.base, currentAveragePricePath)

	parsedURL, err := url.Parse(u)
	if err != nil {
		return 0, err
	}
	r, _ := http.NewRequest(http.MethodGet, parsedURL.String()+fmt.Sprintf("?symbol=%s", symbol), nil)
	r.Header.Set(apiKeyHeader, c.apiKey)

	resp, err := c.h.Do(r)
	if err != nil {
		return 0, err
	}
	var capResp CurrentAveragePriceResponse
	b, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err := json.Unmarshal(b, &capResp); err != nil {
		return 0, err
	}
	if resp.StatusCode != http.StatusOK {
		return 0, parseError(b)
	}

	fl, err := strconv.ParseFloat(capResp.Price, 64)
	if err != nil {
		return 0, err
	}
	return fl, nil
}

func (c Client) Order(req OrderRequest) (*OrderResponse, error) {
	u := fmt.Sprintf("%s/%s", c.base, orderPath)

	parsedURL, err := url.Parse(u)
	if err != nil {
		return nil, err
	}
	fmt.Println(c.createURL(req, parsedURL))
	r, _ := http.NewRequest(http.MethodGet, c.createURL(req, parsedURL), nil)
	r.Header.Set(apiKeyHeader, c.apiKey)

	var order OrderResponse

	resp, err := c.h.Do(r)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, parseError(body)
	}
	if err := json.Unmarshal(body, &order); err != nil {
		return nil, err
	}

	return &order, nil
}

func parseError(b []byte) error {
	var bErr BinanceError
	if err := json.Unmarshal(b, &bErr); err != nil {
		return err
	}
	return bErr
}

func (c Client) createURL(req RequestInterface, parsedURL *url.URL) string {
	q := &url.Values{}
	req.EmbedData(q)

	h := hmac.New(sha256.New, c.secretKey)
	h.Write([]byte(q.Encode()))
	sha := hex.EncodeToString(h.Sum(nil))

	parsedURL.RawQuery = q.Encode() + "&signature=" + sha

	return parsedURL.String()
}
