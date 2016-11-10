// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package gitea

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	// ErrInvalidReceiveHook FIXME
	ErrInvalidReceiveHook = errors.New("Invalid JSON payload received over webhook")
)

// Hook a hook is a web hook when one repository changed
type Hook struct {
	ID      int64             `json:"id"`
	Type    string            `json:"type"`
	URL     string            `json:"-"`
	Config  map[string]string `json:"config"`
	Events  []string          `json:"events"`
	Active  bool              `json:"active"`
	Updated time.Time         `json:"updated_at"`
	Created time.Time         `json:"created_at"`
}

// ListRepoHooks list all the hooks of one repository
func (c *Client) ListRepoHooks(user, repo string) ([]*Hook, error) {
	hooks := make([]*Hook, 0, 10)
	return hooks, c.getParsedResponse("GET", fmt.Sprintf("/repos/%s/%s/hooks", user, repo), nil, nil, &hooks)
}

// CreateHookOption options when create a hook
type CreateHookOption struct {
	Type   string            `json:"type" binding:"Required"`
	Config map[string]string `json:"config" binding:"Required"`
	Events []string          `json:"events"`
	Active bool              `json:"active"`
}

// CreateRepoHook create one hook with options
func (c *Client) CreateRepoHook(user, repo string, opt CreateHookOption) (*Hook, error) {
	body, err := json.Marshal(&opt)
	if err != nil {
		return nil, err
	}
	h := new(Hook)
	return h, c.getParsedResponse("POST", fmt.Sprintf("/repos/%s/%s/hooks", user, repo), jsonHeader, bytes.NewReader(body), h)
}

// EditHookOption options when modify one hook
type EditHookOption struct {
	Config map[string]string `json:"config"`
	Events []string          `json:"events"`
	Active *bool             `json:"active"`
}

// EditRepoHook modify one hook with hook id and options
func (c *Client) EditRepoHook(user, repo string, id int64, opt EditHookOption) error {
	body, err := json.Marshal(&opt)
	if err != nil {
		return err
	}
	_, err = c.getResponse("PATCH", fmt.Sprintf("/repos/%s/%s/hooks/%d", user, repo, id), jsonHeader, bytes.NewReader(body))
	return err
}

// DeleteRepoHook delete one hook with hook id
func (c *Client) DeleteRepoHook(user, repo string, id int64) error {
	_, err := c.getResponse("DELETE", fmt.Sprintf("/repos/%s/%s/hooks/%d", user, repo, id), nil, nil)
	return err
}

// Payloader payload is some part of one hook
type Payloader interface {
	SetSecret(string)
	JSONPayload() ([]byte, error)
}

// PayloadUser FIXME
type PayloadUser struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	UserName string `json:"username"`
}

// PayloadCommit FIXME: consider use same format as API when commits API are added.
type PayloadCommit struct {
	ID        string       `json:"id"`
	Message   string       `json:"message"`
	URL       string       `json:"url"`
	Author    *PayloadUser `json:"author"`
	Committer *PayloadUser `json:"committer"`
	Timestamp time.Time    `json:"timestamp"`
}

var (
	_ Payloader = &CreatePayload{}
	_ Payloader = &PushPayload{}
	_ Payloader = &PullRequestPayload{}
)

// _________                        __
// \_   ___ \_______   ____ _____ _/  |_  ____
// /    \  \/\_  __ \_/ __ \\__  \\   __\/ __ \
// \     \____|  | \/\  ___/ / __ \|  | \  ___/
//  \______  /|__|    \___  >____  /__|  \___  >
//         \/             \/     \/          \/

// CreatePayload FIXME
type CreatePayload struct {
	Secret  string      `json:"secret"`
	Ref     string      `json:"ref"`
	RefType string      `json:"ref_type"`
	Repo    *Repository `json:"repository"`
	Sender  *User       `json:"sender"`
}

// SetSecret FIXME
func (p *CreatePayload) SetSecret(secret string) {
	p.Secret = secret
}

// JSONPayload return payload information
func (p *CreatePayload) JSONPayload() ([]byte, error) {
	return json.MarshalIndent(p, "", "  ")
}

// ParseCreateHook parses create event hook content.
func ParseCreateHook(raw []byte) (*CreatePayload, error) {
	hook := new(CreatePayload)
	if err := json.Unmarshal(raw, hook); err != nil {
		return nil, err
	}

	// it is possible the JSON was parsed, however,
	// was not from Gogs (maybe was from Bitbucket)
	// So we'll check to be sure certain key fields
	// were populated
	switch {
	case hook.Repo == nil:
		return nil, ErrInvalidReceiveHook
	case len(hook.Ref) == 0:
		return nil, ErrInvalidReceiveHook
	}
	return hook, nil
}

// __________             .__
// \______   \__ __  _____|  |__
//  |     ___/  |  \/  ___/  |  \
//  |    |   |  |  /\___ \|   Y  \
//  |____|   |____//____  >___|  /
//                      \/     \/

// PushPayload represents a payload information of push event.
type PushPayload struct {
	Secret     string           `json:"secret"`
	Ref        string           `json:"ref"`
	Before     string           `json:"before"`
	After      string           `json:"after"`
	CompareURL string           `json:"compare_url"`
	Commits    []*PayloadCommit `json:"commits"`
	Repo       *Repository      `json:"repository"`
	Pusher     *User            `json:"pusher"`
	Sender     *User            `json:"sender"`
}

// SetSecret FIXME
func (p *PushPayload) SetSecret(secret string) {
	p.Secret = secret
}

// JSONPayload FIXME
func (p *PushPayload) JSONPayload() ([]byte, error) {
	return json.MarshalIndent(p, "", "  ")
}

// ParsePushHook parses push event hook content.
func ParsePushHook(raw []byte) (*PushPayload, error) {
	hook := new(PushPayload)
	if err := json.Unmarshal(raw, hook); err != nil {
		return nil, err
	}

	switch {
	case hook.Repo == nil:
		return nil, ErrInvalidReceiveHook
	case len(hook.Ref) == 0:
		return nil, ErrInvalidReceiveHook
	}
	return hook, nil
}

// Branch returns branch name from a payload
func (p *PushPayload) Branch() string {
	return strings.Replace(p.Ref, "refs/heads/", "", -1)
}

// .___
// |   | ______ ________ __   ____
// |   |/  ___//  ___/  |  \_/ __ \
// |   |\___ \ \___ \|  |  /\  ___/
// |___/____  >____  >____/  \___  >
//          \/     \/            \/

// HookIssueAction FIXME
type HookIssueAction string

const (
	// HookIssueOpened opened
	HookIssueOpened HookIssueAction = "opened"
	// HookIssueClosed closed
	HookIssueClosed HookIssueAction = "closed"
	// HookIssueReOpened reopened
	HookIssueReOpened HookIssueAction = "reopened"
	// HookIssueEdited edited
	HookIssueEdited HookIssueAction = "edited"
	// HookIssueAssigned assigned
	HookIssueAssigned HookIssueAction = "assigned"
	// HookIssueUnassigned unassigned
	HookIssueUnassigned HookIssueAction = "unassigned"
	// HookIssueLabelUpdated label_updated
	HookIssueLabelUpdated HookIssueAction = "label_updated"
	// HookIssueLabelCleared label_cleared
	HookIssueLabelCleared HookIssueAction = "label_cleared"
	// HookIssueSynchronized synchronized
	HookIssueSynchronized HookIssueAction = "synchronized"
)

// ChangesFromPayload FIXME
type ChangesFromPayload struct {
	From string `json:"from"`
}

// ChangesPayload FIXME
type ChangesPayload struct {
	Title *ChangesFromPayload `json:"title,omitempty"`
	Body  *ChangesFromPayload `json:"body,omitempty"`
}

// __________      .__  .__    __________                                     __
// \______   \__ __|  | |  |   \______   \ ____  ________ __   ____   _______/  |_
//  |     ___/  |  \  | |  |    |       _// __ \/ ____/  |  \_/ __ \ /  ___/\   __\
//  |    |   |  |  /  |_|  |__  |    |   \  ___< <_|  |  |  /\  ___/ \___ \  |  |
//  |____|   |____/|____/____/  |____|_  /\___  >__   |____/  \___  >____  > |__|
//                                     \/     \/   |__|           \/     \/

// PullRequestPayload represents a payload information of pull request event.
type PullRequestPayload struct {
	Secret      string          `json:"secret"`
	Action      HookIssueAction `json:"action"`
	Index       int64           `json:"number"`
	Changes     *ChangesPayload `json:"changes,omitempty"`
	PullRequest *PullRequest    `json:"pull_request"`
	Repository  *Repository     `json:"repository"`
	Sender      *User           `json:"sender"`
}

// SetSecret FIXME
func (p *PullRequestPayload) SetSecret(secret string) {
	p.Secret = secret
}

// JSONPayload FIXME
func (p *PullRequestPayload) JSONPayload() ([]byte, error) {
	return json.MarshalIndent(p, "", "  ")
}
