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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"

	"github.com/orijtech/otils"

	"github.com/odeke-em/go-uuid"
)

type Attachment struct {
	ID int64 `json:"id,omitempty"`

	CreatedAt   *otils.NullableTime  `json:"created_at"`
	DownloadURL otils.NullableString `json:"download_url"`

	// Host is a read-only value.
	// Valid values are asana, dropbox, gdrive and box.
	Host otils.NullableString `json:"host"`

	Name otils.NullableString `json:"name"`

	// Parent contains the information of the
	// task that this attachment is attached to.
	Parent *NamedAndIDdEntity `json:"parent,omitempty"`

	ViewURL otils.NullableString `json:"view_url,omitempty"`
}

var (
	errEmptyAttachmentID = errors.New("expecting a non-empty attachmentID")
	errNoAttachment      = errors.New("no attachment was received")
)

func (c *Client) FindAttachmentByID(attachmentID string) (*Attachment, error) {
	attachmentID = strings.TrimSpace(attachmentID)
	if attachmentID == "" {
		return nil, errEmptyAttachmentID
	}
	fullURL := fmt.Sprintf("%s/attachments/%s", baseURL, attachmentID)
	req, _ := http.NewRequest("GET", fullURL, nil)
	slurp, _, err := c.doAuthReqThenSlurpBody(req)
	if err != nil {
		return nil, err
	}
	return parseOutAttachmentFromData(slurp)
}

type AttachmentWrap struct {
	Attachment *Attachment `json:"data"`
}

func parseOutAttachmentFromData(blob []byte) (*Attachment, error) {
	aWrap := new(AttachmentWrap)
	if err := json.Unmarshal(blob, aWrap); err != nil {
		return nil, err
	}
	if aWrap.Attachment != nil {
		return aWrap.Attachment, nil
	}

	return nil, errNoAttachment
}

type AttachmentUpload struct {
	Body   io.Reader `json:"-"`
	TaskID string    `json:"task_id"`
	Name   string    `json:"name"`
}

func (au *AttachmentUpload) nonBlankFilename() string {
	if au.Name != "" {
		return au.Name
	}
	return uuid.NewRandom().String()
}

var errNilBody = errors.New("expecting a non-nil body")

func (au *AttachmentUpload) Validate() error {
	if au == nil || au.Body == nil {
		return errNilBody
	}
	if strings.TrimSpace(au.TaskID) == "" {
		return errEmptyTaskID
	}
	return nil
}

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func escapeQuotes(s string) string {
	return quoteEscaper.Replace(s)
}

// UploadAtatchment uploads an attachment to a specific task.
// Its fields: TaskID and Body must be set otherwise it will return an error.
func (c *Client) UploadAttachment(au *AttachmentUpload) (*Attachment, error) {
	contentType, _, err := fDetectContentType(au.Body)
	if err != nil {
		return nil, err
	}

	body := &bytes.Buffer{}

	writer := multipart.NewWriter(body)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="file"; type="%s"; filename="%s"`,
			escapeQuotes(contentType), escapeQuotes(au.nonBlankFilename())))
	h.Set("Content-Type", contentType)
	fw, err := writer.CreatePart(h)
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(fw, au.Body)
	if err != nil {
		return nil, err
	}

	err = writer.WriteField("type", contentType)
	if err != nil {
		return nil, err
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	/* if err := au.Validate(); err != nil {
		return nil, err
	}

	// Step 1. Try to determine the contentType.
	// Step 2:
	// Initiate and then make the upload.
	// prc, pwc := io.Pipe()
	var b bytes.Buffer
	mpartW := multipart.NewWriter(&b)
	// go func() {
	// 	defer func() {
	// 		_ = mpartW.Close()
	// 		_ = pwc.Close()
	// 	}()

	// 	formFile, err := mpartW.CreateFormFile("file", au.nonBlankFilename())
	// 	if err != nil {
	// 		return
	// 	}
	// 	writeStringField(mpartW, "Type", contentType)
	// 	// writeStringField(mpartW, "name", au.Name)
	// 	_, _ = io.Copy(formFile, body)

	// }()
	var fw io.Writer
	if x, ok := body.(io.Closer); ok {
		defer x.Close()
	}
	writeStringField(mpartW, "Type", contentType)
	// Add an image file
	if x, ok := body.(*os.File); ok {
		if fw, err = mpartW.CreateFormFile("file", x.Name()); err != nil {
			return nil, err
		}
	} else {
		// Add other fields
		if fw, err = mpartW.CreateFormField("file"); err != nil {
			return nil, err
		}
	}
	if _, err = io.Copy(fw, body); err != nil {
		return nil, err
	}
	*/
	fullURL, err := url.Parse(fmt.Sprintf("%s/tasks/%s/attachments", baseURL, au.TaskID))
	if err != nil {
		return nil, err
	}
	query := fullURL.Query()
	query.Add("opt_fields", "this.name,this.view_url")
	fullURL.RawQuery = query.Encode()
	req, err := http.NewRequest("POST", fullURL.String(), bytes.NewReader(body.Bytes()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	// req.Header.Set("Content-Type", "multipart/form-data")
	slurp, _, err := c.doAuthReqThenSlurpBody(req)
	if err != nil {
		return nil, err
	}
	return parseOutAttachmentFromData(slurp)
}

type AttachmentsPage struct {
	Attachments []*Attachment `json:"data"`
}

// ListAllAttachmentsForTask retrieves all the attachments for the taskID provided.
func (c *Client) ListAllAttachmentsForTask(taskID string) (*AttachmentsPage, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, errEmptyTaskID
	}
	fullURL := fmt.Sprintf("%s/tasks/%s/attachments", baseURL, taskID)
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, err
	}

	slurp, _, err := c.doAuthReqThenSlurpBody(req)
	if err != nil {
		return nil, err
	}

	apage := new(AttachmentsPage)
	if err := json.Unmarshal(slurp, apage); err != nil {
		return nil, err
	}
	return apage, nil
}

func writeStringField(w *multipart.Writer, key, value string) {
	fw, err := w.CreateFormField(key)
	if err == nil {
		_, _ = io.WriteString(fw, value)
	}
}

func fDetectContentType(r io.Reader) (string, io.Reader, error) {
	if r == nil {
		return "", nil, errNilBody
	}

	seeker, seekable := r.(io.Seeker)
	sniffBuf := make([]byte, 512)
	n, err := io.ReadAtLeast(r, sniffBuf, 1)
	if err != nil {
		return "", nil, err
	}

	contentType := http.DetectContentType(sniffBuf)
	needsRepad := !seekable
	if seekable {
		if _, err = seeker.Seek(int64(-n), io.SeekCurrent); err != nil {
			// Since we failed to rewind it, mark it as needing repad
			needsRepad = true
		}
	}

	if needsRepad {
		r = io.MultiReader(bytes.NewReader(sniffBuf), r)
	}

	return contentType, r, nil
}
