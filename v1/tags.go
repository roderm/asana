package asana

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/orijtech/otils"
)

type Tag struct {
	NamedAndIDdEntity
	Color        string     `json:"color"`
	PermalinkURL string     `json:"permalink_url"`
	Workspace    *Workspace `json:"Workspace"`
}

type TagsPage struct {
	Tags []*Tag `json:"data"`
	Err  error
}

type tagsPager struct {
	TagsPage

	NextPage *pageToken `json:"next_page,omitempty"`
}

type tagResultWrap struct {
	Tag *Tag `json:"data"`
}

type CreateTagRequest struct {
	Color       string `json:"color"`
	Name        string `json:"name"`
	WorkspaceID string `json:"workspace"`
}

func (c *Client) CreateTag(t CreateTagRequest) (*Tag, error) {
	// This endpoint takes in url-encoded data
	qs, err := otils.ToURLValues(t)
	if err != nil {
		return nil, err
	}

	for _, field := range readOnlyFields {
		qs.Del(field)
	}

	fullURL := fmt.Sprintf("%s/tags", baseURL)
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
	return parseOutTagFromData(slurp)
}
func parseOutTagFromData(blob []byte) (*Tag, error) {
	wrap := new(tagResultWrap)
	if err := json.Unmarshal(blob, wrap); err != nil {
		return nil, err
	}
	return wrap.Tag, nil
}
func (c *Client) GetTag(tagID string) (*Tag, error) {
	fullURL := fmt.Sprintf("%s%s/%s", baseURL, "/tags", tagID)
	req, _ := http.NewRequest("GET", fullURL, nil)
	slurp, _, err := c.doAuthReqThenSlurpBody(req)
	if err != nil {
		return nil, err
	}
	page := new(TagsPage)
	if err := json.Unmarshal(slurp, page); err != nil {
		if err != nil {
			return nil, err
		}
	}
	if tag := page.Tags[0]; tag != nil {
		return tag, nil
	}
	return nil, fmt.Errorf("No tag found with ID %s", tagID)
}
func (c *Client) ListAllTags() (pagesChan chan *TagsPage, cancelChan chan<- bool, err error) {
	return c.pageForTags("/tags")
}
func (c *Client) pageForTags(path string) (pagesChan chan *TagsPage, cancelChan chan<- bool, err error) {
	pagesChan = make(chan *TagsPage)
	cancelChan = make(chan bool, 1)

	go func() {
		defer close(pagesChan)

		for {
			fullURL := fmt.Sprintf("%s%s", baseURL, path)
			req, _ := http.NewRequest("GET", fullURL, nil)
			slurp, _, err := c.doAuthReqThenSlurpBody(req)

			if err != nil {
				pagesChan <- &TagsPage{Err: err}
				return
			}

			pager := new(tagsPager)
			if err := json.Unmarshal(slurp, pager); err != nil {
				pager.Err = err
			}

			tagsPage := pager.TagsPage
			pagesChan <- &tagsPage

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
