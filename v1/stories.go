package asana

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/orijtech/otils"
)

type Story struct {
	Text     string `json:"text,omitempty"`
	HtmlText string `json:"html_text,omitempty"`
	Pinned   bool   `json:"is_pinned,omitempty"`
	Sticker  string `json:"sticker_name,omitempty"`
}
type storyResultWrap struct {
	Story *Story `json:"data"`
}
type CreateStoryRequest struct {
	TaskID   string
	Text     string `json:"text,omitempty"`
	HtmlText string `json:"html_text,omitempty"`
	Pinned   bool   `json:"is_pinned,omitempty"`
	Sticker  string `json:"sticker_name,omitempty"`
}

func (c *Client) CreateStory(s *CreateStoryRequest) (*Story, error) {
	// This endpoint takes in url-encoded data
	qs, err := otils.ToURLValues(s)
	if err != nil {
		return nil, err
	}

	for _, field := range readOnlyFields {
		qs.Del(field)
	}

	fullURL := fmt.Sprintf("%s/tasks/%s/stories", baseURL, s.TaskID)
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
	return parseOutStoryFromData(slurp)
}
func parseOutStoryFromData(blob []byte) (*Story, error) {
	wrap := new(storyResultWrap)
	if err := json.Unmarshal(blob, wrap); err != nil {
		return nil, err
	}
	return wrap.Story, nil
}
