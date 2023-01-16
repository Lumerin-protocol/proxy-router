package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"gitlab.com/TitanInd/hashrouter/interfaces"
)

const (
	ContentTypeApplicationJSON = "application/json"
)

type ApiClient struct {
	baseUrl *url.URL
}

func NewApiClient(baseUrlStr string) (*ApiClient, error) {
	baseUrl, err := url.Parse(baseUrlStr)
	if err != nil {
		return nil, err
	}
	return &ApiClient{
		baseUrl: baseUrl,
	}, nil
}

func (a *ApiClient) Health() error {
	resp, err := http.Get(a.baseUrl.JoinPath("/healthcheck").String())
	if err != nil {
		return err
	}
	if err := checkStatus(resp); err != nil {
		return err
	}

	return nil
}

func (a *ApiClient) GetMiners() (*MinersResponse, error) {
	return get[MinersResponse](a.baseUrl, "/miners")
}

func (a *ApiClient) GetMiner(ID string) (*Miner, error) {
	return get[Miner](a.baseUrl, fmt.Sprintf("/miners/%s", ID))
}

func (a *ApiClient) GetContracts() (*[]Contract, error) {
	return get[[]Contract](a.baseUrl, "/contracts")
}

func (a *ApiClient) GetContract(ID string) (*Contract, error) {
	return get[Contract](a.baseUrl, fmt.Sprintf("/contracts/%s", ID))
}

func (a *ApiClient) PostContract(dest interfaces.IDestination, hrGHS int, duration time.Duration) error {
	qs := url.Values{}
	qs.Add("dest", dest.String())
	qs.Add("hrGHS", fmt.Sprintf("%d", hrGHS))
	qs.Add("duration", duration.String())
	_, err := post[any](a.baseUrl, "/contracts", qs)
	return err
}

func (a *ApiClient) PostContractDest(address string, dest interfaces.IDestination) error {
	qs := url.Values{}
	qs.Add("dest", dest.String())
	_, err := post[any](a.baseUrl, fmt.Sprintf("/contracts/%s/dest", address), qs)
	return err
}

func checkStatus(resp *http.Response) error {
	if resp.StatusCode != http.StatusOK {
		b, err := io.ReadAll(resp.Body)
		var errStr string
		if err != nil {
			errStr = err.Error()
		} else {
			errStr = string(b)
		}
		return fmt.Errorf("response status code(%d): %s", resp.StatusCode, errStr)
	}
	return nil
}

func post[T any](baseUrl *url.URL, path string, qs url.Values) (*T, error) {
	targetUrl := baseUrl.JoinPath(path)
	targetUrl.RawQuery = qs.Encode()
	resp, err := http.Post(targetUrl.String(), ContentTypeApplicationJSON, nil)
	if err != nil {
		return nil, err
	}
	if err := checkStatus(resp); err != nil {
		return nil, err
	}

	if resp.ContentLength == 0 {
		return nil, nil
	}

	var response T
	err = unmarshal(resp.Body, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

func get[T any](baseUrl *url.URL, path string) (*T, error) {
	resp, err := http.Get(baseUrl.JoinPath(path).String())
	if err != nil {
		return nil, err
	}
	if err := checkStatus(resp); err != nil {
		return nil, err
	}

	var response T
	err = unmarshal(resp.Body, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

func unmarshal[T any](body io.ReadCloser, data T) error {
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	return json.Unmarshal(bodyBytes, data)
}
