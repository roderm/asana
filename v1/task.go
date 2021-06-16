// Copyright 2017 orijtech. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package asana

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/orijtech/otils"
)

type Task struct {
	NamedAndIDdEntity
	// ID          int64              `json:"id,omitempty"`
	Assignee    *NamedAndIDdEntity `json:"assignee,omitempty"`
	CreatedAt   *time.Time         `json:"created_at,omitempty"`
	Completed   bool               `json:"completed,omitempty"`
	CompletedAt *time.Time         `json:"completed_at,omitempty"`

	AssigneeStatus AssigneeStatus `json:"assignee_status,omitempty"`

	CustomFields []CustomField `json:"custom_fields,omitempty"`

	DueOn *YYYYMMDD  `json:"due_on,omitempty"`
	DueAt *time.Time `json:"due_at,omitempty"`

	Metadata Metadata `json:"external,omitempty"`

	Followers []*User `json:"followers,omitempty"`

	HeartedByMe bool                 `json:"hearted,omitempty"`
	Hearts      []*NamedAndIDdEntity `json:"hearts,omitempty"`
	HeartCount  int64                `json:"num_hearts,omitempty"`
	ModifiedAt  *time.Time           `json:"modified_at"`

	// Name string `json:"name,omitempty"`

	Notes string `json:"notes,omitempty"`

	Projects   []*Project `json:"projects,omitempty"`
	ParentTask *Task      `json:"parent,omitempty"`

	Workspace *NamedAndIDdEntity `json:"workspace,omitempty"`

	Memberships []*Membership `json:"memberships,omitempty"`

	Tags []*Tag `json:"tags,omitempty"`
}

type NamedAndIDdEntity struct {
	Name string `json:"name"`
	ID   string `json:"gid"`
}

type Membership struct {
	Project *NamedAndIDdEntity `json:"project,omitempty"`
	Section *NamedAndIDdEntity `json:"section,omitempty"`
}

type AssigneeStatus string

const (
	StatusInbox    AssigneeStatus = "inbox"
	StatusLater    AssigneeStatus = "later"
	StatusToday    AssigneeStatus = "today"
	StatusUpcoming AssigneeStatus = "upcoming"
)

const (
	defaultAssigneeStatus = StatusInbox
)

func (as AssigneeStatus) String() string {
	str := string(as)
	if str != "" {
		return str
	}
	return string(defaultAssigneeStatus)
}

func (as AssigneeStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(as))
}

// type CustomField map[string]interface{}

type Metadata map[string]interface{}

type YYYYMMDD struct {
	sync.RWMutex

	YYYY int64
	MM   int64
	DD   int64

	str string
}

var _ json.Marshaler = (*YYYYMMDD)(nil)
var _ json.Unmarshaler = (*YYYYMMDD)(nil)

func (ymd *YYYYMMDD) UnmarshalJSON(b []byte) error {
	// Format of data is: 2012-03-26
	unquoted, err := strconv.Unquote(string(b))
	if err != nil {
		return err
	}
	splits := strings.Split(unquoted, "-")
	if len(splits) < 3 {
		return errors.New("expecting YYYY-MM-DD")
	}

	var intified []int64
	for _, split := range splits {
		it, err := intifyIt(split)
		if err != nil {
			return err
		}
		intified = append(intified, it)
	}

	ymd.YYYY = intified[0]
	ymd.MM = intified[1]
	ymd.DD = intified[2]
	return nil
}

func (ymd *YYYYMMDD) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(ymd.String()))
}

func (ymd *YYYYMMDD) String() string {
	if ymd == nil {
		return ""
	}
	ymd.Lock()
	defer ymd.Unlock()
	if ymd.str == "" {
		ymd.str = fmt.Sprintf("%d-%d-%d", ymd.YYYY, ymd.MM, ymd.DD)
	}
	return ymd.str
}

func intifyIt(st string) (int64, error) {
	return strconv.ParseInt(st, 10, 64)
}

func (c *Client) doAuthReqThenSlurpBody(req *http.Request) ([]byte, http.Header, error) {
	req.Header.Set("Authorization", c.personalAccessTokenAuthValue())
	res, err := c.httpClient().Do(req)
	if err != nil {
		return nil, nil, err
	}
	if res.Body != nil {
		defer res.Body.Close()
	}

	if !otils.StatusOK(res.StatusCode) {
		errMsg := res.Status
		if res.Body != nil {
			slurp, _ := ioutil.ReadAll(res.Body)
			if len(slurp) > 0 {
				errMsg = string(slurp)
			}
		}
		return nil, res.Header, &HTTPError{msg: errMsg, code: res.StatusCode}
	}

	slurp, err := ioutil.ReadAll(res.Body)
	return slurp, res.Header, err
}

var readOnlyFields = []string{
	"num_hearts",
}

type taskResultWrap struct {
	Task *Task `json:"data"`
}

func (c *Client) CreateTask(t *TaskRequest) (*Task, error) {
	type myTaskRequest struct {
		Data *TaskRequest `json:"data"`
	}
	t.HeartCount = nil
	queryStr, err := json.Marshal(&myTaskRequest{
		Data: t,
	})
	fullURL := fmt.Sprintf("%s/tasks", baseURL)
	reqBody := string(queryStr)
	fmt.Println(reqBody)
	req, err := http.NewRequest("POST", fullURL, strings.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	slurp, _, err := c.doAuthReqThenSlurpBody(req)
	if err != nil {
		return nil, err
	}
	return parseOutTaskFromData(slurp)
}

func parseOutTaskFromData(blob []byte) (*Task, error) {
	wrap := new(taskResultWrap)
	if err := json.Unmarshal(blob, wrap); err != nil {
		return nil, err
	}
	return wrap.Task, nil
}

type TaskResultPage struct {
	Tasks []*Task `json:"data"`
	Err   error
}

type taskPager struct {
	TaskResultPage

	NextPage *pageToken `json:"next_page,omitempty"`
}

type TaskRequest struct {
	Page        int        `json:"page,omitempty"`
	Limit       int        `json:"limit,omitempty"`
	MaxRetries  int        `json:"max_retries,omitempty"`
	Assignee    string     `json:"assignee,omitempty"`
	ProjectID   string     `json:"project,omitempty"`
	Workspace   string     `json:"workspace,omitempty"`
	ID          int64      `json:"id,omitempty"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
	Completed   bool       `json:"completed,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	AssigneeStatus AssigneeStatus `json:"assignee_status,omitempty"`

	CustomFields map[string]interface{} `json:"custom_fields,omitempty"`

	DueOn *YYYYMMDD  `json:"due_on,omitempty"`
	DueAt *time.Time `json:"due_at,omitempty"`

	Metadata Metadata `json:"external,omitempty"`

	Followers []UserID `json:"followers,omitempty"`

	HeartedByMe bool       `json:"hearted,omitempty"`
	Hearts      []*User    `json:"hearts,omitempty"`
	HeartCount  *int64     `json:"num_hearts,omitempty"`
	ModifiedAt  *time.Time `json:"modified_at,omitempty"`

	Name string `json:"name,omitempty"`

	Notes string `json:"notes,omitempty"`

	Projects   []string `json:"projects,omitempty"`
	ParentTask string   `json:"parent,omitempty"`

	Memberships []*Membership `json:"memberships,omitempty"`

	Tags []string `json:"tags,omitempty"`
}

type listTaskWrap struct {
	Tasks []*Task `json:"data"`
}

func (c *Client) ListAllMyTasks() (resultsChan chan *TaskResultPage, cancelChan chan<- bool, err error) {
	cancelChan = make(chan bool)
	treq, err := c.ListMyTasks(nil)
	return treq, cancelChan, err
}

const defaultTaskLimit = 20

func (treq *TaskRequest) fillWithDefaults() {
	if treq == nil {
		return
	}
	if treq.Limit <= 0 {
		treq.Limit = defaultTaskLimit
	}
}

func (c *Client) ListMyTasks(treq *TaskRequest) (chan *TaskResultPage, error) {
	theReq := new(TaskRequest)
	if treq != nil {
		*theReq = *treq
	}
	theReq.Assignee = MeAsUser
	theReq.fillWithDefaults()
	qs, err := otils.ToURLValues(theReq)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/tasks?%s", qs.Encode())
	pageChan, _, err := c.doTasksPaging(path)
	return pageChan, err
}

type WorkspacePage struct {
	Err        error
	Workspaces []*Workspace `json:"data,omitempty"`

	NextPage *pageToken `json:"next_page,omitempty"`
}

type pageToken struct {
	Offset string `json:"offset"`
	Path   string `json:"path"`
	URI    string `json:"uri"`
}

func (c *Client) ListMyWorkspaces() (chan *WorkspacePage, error) {
	wspChan := make(chan *WorkspacePage)
	go func() {
		defer close(wspChan)

		path := "/workspaces"
		for {
			fullURL := fmt.Sprintf("%s%s", baseURL, path)
			req, _ := http.NewRequest("GET", fullURL, nil)
			slurp, _, err := c.doAuthReqThenSlurpBody(req)
			if err != nil {
				wspChan <- &WorkspacePage{Err: err}
				return
			}

			page := new(WorkspacePage)
			if err := json.Unmarshal(slurp, page); err != nil {
				page.Err = err
			}

			wspChan <- page

			if np := page.NextPage; np != nil && np.Path == "" {
				path = np.Path
			} else {
				// End of this pagination
				break
			}
		}
	}()

	return wspChan, nil
}

var errEmptyTaskID = errors.New("expecting a non-empty taskID")

func (c *Client) FindTaskByID(taskID string) (*Task, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, errEmptyTaskID
	}
	fullURL := fmt.Sprintf("%s/tasks/%s", baseURL, taskID)
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, err
	}
	slurp, _, err := c.doAuthReqThenSlurpBody(req)
	if err != nil {
		return nil, err
	}
	return parseOutTaskFromData(slurp)
}

var errEmptyProjectID = errors.New("expecting a non-empty projectID")

func (c *Client) ListTasksForProject(treq *TaskRequest) (resultsChan chan *TaskResultPage, cancelChan chan<- bool, err error) {
	path := fmt.Sprintf("/projects/%s/tasks", treq.ProjectID)
	return c.doTasksPaging(path)
}

func (c *Client) doTasksPaging(path string) (resultsChan chan *TaskResultPage, cancelChan chan<- bool, err error) {
	tasksPageChan := make(chan *TaskResultPage)
	cancelChan = make(chan bool, 1)
	fields := []string{
		"name",
		"assignee",
		"created_at",
		"completed",
		"completed_at",
		"assignee_status",
		"custom_fields",
		"due_on",
		"due_at",
		"external",
		"followers.name",
		"followers.email",
		"hearted",
		"hearts",
		"num_hearts",
		"modified_at",
		"tags.name",
		"tags.color",
		"projects.name",
		"projects.custom_fields",
	}
	go func() {
		defer close(tasksPageChan)

		for {
			uri, _ := url.Parse(fmt.Sprintf("%s%s", baseURL, path))
			query := uri.Query()
			if len(fields) > 0 {
				query.Add("opt_fields", fmt.Sprintf("this.%s", strings.Join(fields[:], ",this.")))
			}
			uri.RawQuery = query.Encode()
			fullURL := uri.String()
			fmt.Println("url:", fullURL)
			req, err := http.NewRequest("GET", fullURL, nil)
			if err != nil {
				tasksPageChan <- &TaskResultPage{Err: err}
				return
			}

			slurp, _, err := c.doAuthReqThenSlurpBody(req)
			fmt.Println(string(slurp))
			if err != nil {
				tasksPageChan <- &TaskResultPage{Err: err}
				return
			}

			pager := new(taskPager)
			if err := json.Unmarshal(slurp, pager); err != nil {
				pager.Err = err
			}

			taskPage := pager.TaskResultPage
			tasksPageChan <- &taskPage

			if np := pager.NextPage; np != nil && np.Path == "" {
				path = np.Path
			} else {
				// End of this pagination
				break
			}
		}
	}()

	return tasksPageChan, cancelChan, nil
}

type SearchRequest struct {
	fields map[string]string
}
type SearchOpt func(*SearchRequest)

func WithField(key string, value string) SearchOpt {
	return func(sr *SearchRequest) {
		sr.fields[key] = value
	}
}

func WithCustomFieldIsSet(id string, value bool) SearchOpt {
	return func(sr *SearchRequest) {
		sr.fields[fmt.Sprintf("custom_fields.%s.is_set", id)] = fmt.Sprintf("%t", value)
	}
}
func WithCustomFieldValue(id string, value interface{}) SearchOpt {
	return func(sr *SearchRequest) {
		sr.fields[fmt.Sprintf("custom_fields.%s.value", id)] = value.(string)
	}
}
func WithCustomFieldStarts(id string, value string) SearchOpt {
	return func(sr *SearchRequest) {
		sr.fields[fmt.Sprintf("custom_fields.%s.starts_with", id)] = value
	}
}
func WithCustomFieldEnds(id string, value string) SearchOpt {
	return func(sr *SearchRequest) {
		sr.fields[fmt.Sprintf("custom_fields.%s.ends_with", id)] = value
	}
}

func WithCustomFieldContains(id string, value string) SearchOpt {
	return func(sr *SearchRequest) {
		sr.fields[fmt.Sprintf("custom_fields.%s.contains", id)] = value
	}
}

func WithCustomFieldLess(id string, value float64) SearchOpt {
	return func(sr *SearchRequest) {
		sr.fields[fmt.Sprintf("custom_fields.%s.less_than", id)] = fmt.Sprintf("%f", value)
	}
}

func WithCustomFieldGreater(id string, value float64) SearchOpt {
	return func(sr *SearchRequest) {
		sr.fields[fmt.Sprintf("custom_fields.%s.greater_than", id)] = fmt.Sprintf("%f", value)
	}
}

func (c *Client) SearchTask(workspaceID string, opts ...SearchOpt) (pagesChan chan *TaskResultPage, cancelChan chan<- bool, err error) {
	sr := &SearchRequest{
		fields: make(map[string]string),
	}
	for _, o := range opts {
		o(sr)
	}
	query := make(url.Values)
	for k, v := range sr.fields {
		query.Add(k, v)
	}
	path := fmt.Sprintf("/workspaces/%s/tasks/search?%s", workspaceID, query.Encode())
	fmt.Println("Path:", path)
	return c.doTasksPaging(path)
}

func (c *Client) DeleteTask(taskID string) error {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return errEmptyTaskID
	}
	fullURL := fmt.Sprintf("%s/tasks/%s", baseURL, taskID)
	req, _ := http.NewRequest("DELETE", fullURL, nil)
	_, _, err := c.doAuthReqThenSlurpBody(req)
	return err
}
