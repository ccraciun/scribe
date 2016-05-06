package lego

import (
	log "github.com/Sirupsen/logrus"
	"time"

	// NOTE: Using aggrolite's branch since he's got OAuth. Switch back to
	// github.com/jzelinskie/geddit once OAuth makes it upstream.
	"github.com/aggrolite/geddit"
)

// Reddit is the base structure common to reddit blocks such as channel source/sink and user source
type Reddit struct {
	ClientID     string
	ClientSecret string
	Username     string
	Password     string
}

// RedditSub implements the Source and Sink interface for reddit subreddits.
type RedditSub struct {
	Reddit
	SubReddit string
}

// RedditUser implements the Source interface for reddit users.
type RedditUser struct {
	Reddit
	User    string
	Listing string
}

// Output creates, starts and returns a Pusher Brick to a reddit channel.
func (r *RedditSub) Output() Pusher {
	o, err := r.redditAuth()
	if err != nil {
		// FIXME: reddit often errors out here before throttling with 429 too many requests
		log.WithError(err).Fatal(err)
	}

	p := &subredditSink{
		legoBrickSink: legoBrickSink{newBrick()},
		subreddit:     r.SubReddit,
		session:       o,
	}
	go p.loop()
	return p
}

// Intake creates, starts and returns a Puller Brick from a subreddit.
func (r *RedditSub) Intake() Puller {
	o, err := r.redditAuth()
	if err != nil {
		// FIXME: reddit often errors out here before throttling with 429 too many requests
		log.WithError(err).Fatal(err)
	}

	p := &subredditSource{
		legoBrickSource: legoBrickSource{newBrick()},
		subreddit:       r.SubReddit,
		session:         o,
	}
	go p.loop()
	return p
}

// Intake creates, starts and returns a Puller Brick from a reddit user.
func (r *RedditUser) Intake() Puller {
	o, err := r.redditAuth()
	if err != nil {
		// FIXME: reddit often errors out here before throttling with 429 too many requests
		log.WithError(err).Fatal(err)
	}

	p := &userSource{
		legoBrickSource: legoBrickSource{newBrick()},
		user:            r.User,
		session:         o,
	}
	go p.loop()
	return p
}

func (r *Reddit) redditAuth() (*geddit.OAuthSession, error) {
	log.Debug("Trying to create OAuth session to reddit")
	o, err := geddit.NewOAuthSession(
		r.ClientID,
		r.ClientSecret,
		"scribe lego brick for reddit",
		"",
	)
	o.Throttle(10 * time.Second)
	if err != nil {
		return nil, err
	}
	log.Debug("Created new OAuth session")

	log.Debug("Trying to login to reddit with user ", r.Username)
	// Login using our personal reddit account.
	err = o.LoginAuth(r.Username, r.Password)
	return o, err
}

type subredditSink struct {
	legoBrickSink

	subreddit string
	session   *geddit.OAuthSession
}

type subredditSource struct {
	legoBrickSource

	subreddit string
	session   *geddit.OAuthSession
}

func (p *subredditSink) loop() {
	for {
		select {
		case d, ok := <-p.pipe:
			if !ok {
				return
			}
			x, ok := d.(geddit.NewSubmission)
			if !ok {
				log.Warningf("Trying to submit bad entry %v to reddit", d)
				continue
			}
			_, err := p.session.Submit(&x)
			if err != nil {
				log.WithError(err).Warningf("Submitting of %v failed", x)
			}
		case errc := <-p.closing:
			errc <- p.err
			return
		}
	}
}

// TODO: abstract the loop into legoBrick if possible since we're duplicating code. To do this we
// just need to define/implement a few interfaces for a Fetcher for the pull loop, and a Sender for
// the push loop.
func (p *subredditSource) loop() {
	var pending []interface{}
	var seen = make(map[string]bool)

	var nextFetch time.Time

	opts := geddit.ListingOptions{}

	for {
		var delay time.Duration
		var doFetch <-chan time.Time
		if now := time.Now(); nextFetch.After(now) {
			delay = nextFetch.Sub(now)
		}
		if len(pending) < maxPending {
			doFetch = time.After(delay)
		}

		var first interface{}     // first item to send
		var pipe chan interface{} // nillable channel shadowing struct.pipe
		if len(pending) > 0 {
			first = pending[0]
			pipe = p.pipe
		}

		select {
		case pipe <- first:
			pending = pending[1:]
		case <-doFetch:
			posts, err := p.session.SubredditSubmissions(
				p.subreddit,
				geddit.NewSubmissions,
				opts)
			if err != nil {
				log.Fatal(err)
			}
			nextFetch = time.Now().Add(fetchDelay)

			for _, d := range posts {
				if !seen[d.FullID] {
					pending = append(pending, d)
					seen[d.FullID] = true
				}
			}
		case errc := <-p.closing:
			errc <- p.err
			close(p.pipe)
			return
		}
	}
}

type userSource struct {
	legoBrickSource

	user    string
	listing string
	session *geddit.OAuthSession
}

func (p *userSource) loop() {
	var pending []interface{}
	var seen = make(map[string]bool)

	var nextFetch time.Time

	opts := geddit.ListingOptions{}

	for {
		var delay time.Duration
		var doFetch <-chan time.Time
		if now := time.Now(); nextFetch.After(now) {
			delay = nextFetch.Sub(now)
		}
		if len(pending) < maxPending {
			doFetch = time.After(delay)
		}

		var first interface{}     // first item to send
		var pipe chan interface{} // nillable channel shadowing struct.pipe
		if len(pending) > 0 {
			first = pending[0]
			pipe = p.pipe
		}

		select {
		case pipe <- first:
			pending = pending[1:]
		case <-doFetch:
			posts, err := p.session.Listing(
				p.user,
				p.listing,
				geddit.NewSubmissions,
				opts)
			if err != nil {
				log.Fatal(err)
			}
			nextFetch = time.Now().Add(fetchDelay)

			for _, d := range posts {
				if !seen[d.FullID] {
					pending = append(pending, d)
					seen[d.FullID] = true
				}
			}
		case errc := <-p.closing:
			errc <- p.err
			close(p.pipe)
			return
		}
	}
}
