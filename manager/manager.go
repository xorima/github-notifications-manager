package manager

import (
	"context"
	"github.com/google/go-github/v50/github"
	"github.com/xorima/github-notifications-manager/config"
	"github.com/youshy/logger"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
	"golang.org/x/oauth2"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

var ctx = context.Background()

type Manager struct {
	client *github.Client
	cfg    *config.Config
	log    *zap.SugaredLogger
}

func NewManager(cfg *config.Config) *Manager {
	client := NewGithubClientPAT(ctx, cfg.GithubToken)
	log := logger.NewLogger(logger.INFO, false)
	return &Manager{
		client: client,
		cfg:    cfg,
		log:    log,
	}
}

func (m *Manager) Handle() {
	nots, err := m.ListNotifications()
	if err != nil {
		if len(nots) == 0 {
			m.log.Errorf("No notifications found and error returned: %s", err.Error())
		}
		m.log.Warnf("Error returned %s, but continuing to process", err.Error())
	}
	m.log.Infof("Found %d notifications", len(nots))
	nots = m.ValidateOrg(nots)
	m.log.Infof("Found %d notifications for org %s", len(nots), m.cfg.OrgName)
	nots, _ = m.ValidateRepoConcurrent(nots)
	m.log.Infof("Found %d notifications for repo with appropriate states %s", len(nots), m.cfg.State)
	m.MarkAsReadConcurrent(nots)
}

func (m *Manager) ListNotifications() ([]*github.Notification, error) {
	var response []*github.Notification
	opts := &github.NotificationListOptions{All: false}
	nots, r, err := m.client.Activity.ListNotifications(ctx, opts)
	if err != nil {
		return response, err
	}
	response = append(response, nots...)
	for r.NextPage != 0 {
		opts.Page = r.NextPage
		nots, r, err = m.client.Activity.ListNotifications(ctx, opts)
		if err != nil {
			return response, err
		}
		response = append(response, nots...)
	}
	return response, nil

}

func (m *Manager) ValidateOrg(n []*github.Notification) []*github.Notification {
	if m.cfg.OrgName == "*" {
		return n
	}
	var response []*github.Notification
	for _, v := range n {
		if v.GetRepository().GetOwner().GetLogin() == m.cfg.OrgName {
			response = append(response, v)
		}
	}
	return response
}

func (m *Manager) MarkAsReadConcurrent(n []*github.Notification) {
	var wg sync.WaitGroup
	for _, v := range n {
		wg.Add(1)
		go func(v *github.Notification) {
			defer wg.Done()
			if m.cfg.DryRun {
				m.log.Infof("Would mark %s\t for repo %s \t id: %s as read", v.GetSubject().GetTitle(), v.GetRepository().GetFullName(), v.GetID())
				return
			}
			_, err := m.client.Activity.MarkThreadRead(ctx, v.GetID())
			if err != nil {
				m.log.Errorf("Error marking thread as read %s", err.Error())
			}
			_, err = m.client.Activity.DeleteThreadSubscription(ctx, v.GetID())
			if err != nil {
				m.log.Errorf("Error deleting thread subscription %s", err.Error())
			}

		}(v)
	}
	wg.Wait()
}

func (m *Manager) ValidateRepoConcurrent(n []*github.Notification) ([]*github.Notification, []*github.Notification) {
	var response []*github.Notification
	var NotMatched []*github.Notification
	var wg sync.WaitGroup
	for _, v := range n {
		wg.Add(1)
		go func(v *github.Notification) {
			defer wg.Done()
			if v.GetSubject().GetType() == "PullRequest" {
				pr, _, err := m.client.PullRequests.Get(ctx, v.GetRepository().GetOwner().GetLogin(), v.GetRepository().GetName(), getId(v.GetSubject().GetURL()))
				if err != nil {
					m.log.Errorf("Error getting pr %s for %s", err.Error(), v.GetSubject().GetURL())
				}
				if slices.Contains(m.cfg.GetState(), pr.GetState()) {
					response = append(response, v)
				}
			} else if v.GetSubject().GetType() == "Issue" {
				issue, _, err := m.client.Issues.Get(ctx, v.GetRepository().GetOwner().GetLogin(), v.GetRepository().GetName(), getId(v.GetSubject().GetURL()))
				if err != nil {
					m.log.Errorf("Error getting issue %s for %s", err.Error(), v.GetSubject().GetURL())

				}
				if slices.Contains(m.cfg.GetState(), issue.GetState()) {
					response = append(response, v)
				}
			} else {
				NotMatched = append(NotMatched, v)
			}
		}(v)
	}
	wg.Wait()
	return response, NotMatched
}

func getId(url string) int {
	last := url[strings.LastIndex(url, "/")+1:]
	id, err := strconv.Atoi(last)
	if err != nil {
		panic(err)
	}
	return id
}

func NewGithubClientPAT(ctx context.Context, accessToken string) *github.Client {
	httpClient := newOauthClientAccessToken(ctx, accessToken)
	return github.NewClient(httpClient)
}

func newOauthClientAccessToken(ctx context.Context, accessToken string) *http.Client {
	c := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	return oauth2.NewClient(ctx, c)
}
