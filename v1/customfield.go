package asana

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/orijtech/otils"
)

type CustomField struct {
	NamedAndIDdEntity
	CurrencyCode         string `json:"currency_code,omitempty"`
	CustomeLabel         string `json:"custom_label,omitempty"`
	CustomeLabelPosition string `json:"custom_label_position,omitempty"`
	Description          string `json:"description,omitempty"`
	Enabled              bool   `json:"enabled,omitempty"`
	NumberValue          int    `json:"number_value,omitempty"`

	Type        string `json:"resource_subtype,omitempty"`
	WorkspaceID string `json:"workspace,omitempty"`
	// enum_options
	// format
	// has_notifications_enabled
}

type CustomFieldsPage struct {
	CustomFields []*CustomField `json:"data"`
	Err          error
}

type customFieldsPager struct {
	CustomFieldsPage

	NextPage *pageToken `json:"next_page,omitempty"`
}

type CreateCustomFieldRequest struct {
	CustomField
}

type customFieldResultWrap struct {
	CustomField *CustomField `json:"data"`
}

func parseOutCustomFieldFromData(blob []byte) (*CustomField, error) {
	wrap := new(customFieldResultWrap)
	if err := json.Unmarshal(blob, wrap); err != nil {
		return nil, err
	}
	return wrap.CustomField, nil
}
func (c *Client) CreateCustomField(cf CreateCustomFieldRequest) (*CustomField, error) {
	qs, err := otils.ToURLValues(cf)
	if err != nil {
		return nil, err
	}

	for _, field := range readOnlyFields {
		qs.Del(field)
	}

	fullURL := fmt.Sprintf("%s/custom_fields", baseURL)
	queryStr := qs.Encode()
	req, err := http.NewRequest("POST", fullURL, strings.NewReader(queryStr))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	slurp, _, err := c.doAuthReqThenSlurpBody(req)
	if err != nil {
		return nil, err
	}
	return parseOutCustomFieldFromData(slurp)
}

func (c *Client) GetCustomFields(WorkspaceID string) (pagesChan chan *CustomFieldsPage, cancelChan chan<- bool, err error) {
	return c.pageForCustomFields(fmt.Sprintf("/workspaces/%s/custom_fields", WorkspaceID))
}

func (c *Client) pageForCustomFields(path string) (pagesChan chan *CustomFieldsPage, cancelChan chan<- bool, err error) {
	pagesChan = make(chan *CustomFieldsPage)
	cancelChan = make(chan bool, 1)

	go func() {
		defer close(pagesChan)

		for {
			fullURL := fmt.Sprintf("%s%s", baseURL, path)
			req, _ := http.NewRequest("GET", fullURL, nil)
			slurp, _, err := c.doAuthReqThenSlurpBody(req)

			if err != nil {
				pagesChan <- &CustomFieldsPage{Err: err}
				return
			}

			pager := new(customFieldsPager)
			if err := json.Unmarshal(slurp, pager); err != nil {
				pager.Err = err
			}

			customFieldsPage := pager.CustomFieldsPage
			pagesChan <- &customFieldsPage

			if np := pager.NextPage; np != nil && np.Path == "" {
				path = np.Path
			} else {
				// End of this pagination
				break
			}
		}
	}()
	return pagesChan, cancelChan, nil
}
