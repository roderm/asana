package asana

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func (c *Client) ListAllUsers() (pagesChan chan *UsersPage, cancelChan chan<- bool, err error) {

	return c.pageForUsers("/users")
}

func (c *Client) pageForUsers(path string) (pagesChan chan *UsersPage, cancelChan chan<- bool, err error) {
	pagesChan = make(chan *UsersPage)
	cancelChan = make(chan bool, 1)

	go func() {
		defer close(pagesChan)

		for {
			fullURL := fmt.Sprintf("%s%s", baseURL, path)
			req, _ := http.NewRequest("GET", fullURL, nil)
			slurp, _, err := c.doAuthReqThenSlurpBody(req)

			if err != nil {
				pagesChan <- &UsersPage{Err: err}
				return
			}

			pager := new(usersPager)
			if err := json.Unmarshal(slurp, pager); err != nil {
				pager.Err = err
			}

			UsersPage := pager.UsersPage
			pagesChan <- &UsersPage

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
