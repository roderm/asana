package asana

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Workspace struct {
	NamedAndIDdEntity

	EMailDomains   []string `json:"email_domains"`
	IsOrganisation bool     `json:"is_organization"`
}

func (c *Client) GetWorkspace(WorkspaceId string) (*Workspace, error) {
	fullURL := fmt.Sprintf("%s%s/%s", baseURL, "/workspaces", WorkspaceId)
	req, _ := http.NewRequest("GET", fullURL, nil)
	slurp, _, err := c.doAuthReqThenSlurpBody(req)
	if err != nil {
		return nil, err
	}
	page := new(WorkspacePage)
	if err := json.Unmarshal(slurp, page); err != nil {
		if err != nil {
			return nil, err
		}
	}
	if ws := page.Workspaces[0]; ws != nil {
		return ws, nil
	}
	return nil, fmt.Errorf("No workspace found with ID %s", WorkspaceId)
}
