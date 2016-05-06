package lego

import (
	// "log"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/SlyMarbo/rss"
)

// Feed implements the Source interface for rss/atom feeds.
type Feed struct {
	URL string
}

// Intake creates and returns Puller for the given feed.
func (r *Feed) Intake() Puller {
	f := &feedStruct{
		legoBrickSource: legoBrickSource{newBrick()},
	}
	go f.loop(r.URL)
	return f
}

// feedStruct implements Puller for rss/atom feeds.
type feedStruct struct {
	legoBrickSource

	fetcher *rss.Feed // fetches items
}

func (f *feedStruct) loop(url string) {
	var pending []interface{}
	var seen = make(map[string]bool) // Track seen items by ID for deduping.
	var err error

	log.Debug("Fetching initial batch")
	// Creating the fetcher and fetching the first batch are coupled.
	// if this becomes annoying, look for better rss library.
	f.fetcher, err = rss.Fetch(url)
	if err != nil {
		log.Warning("Got error fetching:", err)
		f.err = err
		time.Sleep(60 * time.Second)
	}
	pending = make([]interface{}, len(f.fetcher.Items))
	for i, d := range f.fetcher.Items {
		pending[i] = d
		seen[d.ID] = true
	}

	for {
		var delay time.Duration
		var doFetch <-chan time.Time
		if now := time.Now(); f.fetcher.Refresh.After(now) {
			delay = f.fetcher.Refresh.Sub(now)
			log.Debugf("Feed source asks for refresh on: %s, waiting %s before update",
				f.fetcher.Refresh, delay)
		}
		if len(pending) < maxPending {
			// fetch only if we're under the limit.
			doFetch = time.After(delay)
		}

		var first interface{}     // first item to send
		var pipe chan interface{} // nillable channel shadowing struct.pipe
		if len(pending) > 0 {
			first = pending[0]
			pipe = f.pipe
		}

		select {
		case pipe <- first:
			pending = pending[1:]
		case <-doFetch:
			log.Debug("Calling Update to fetch from feed")
			err = f.fetcher.Update()
			if err != nil {
				log.Warning("Got error fetching: ", err)
				f.err = err
				time.Sleep(60 * time.Second)
				continue
			}
			for _, d := range f.fetcher.Items {
				if !seen[d.ID] {
					log.Debug("Got unseen item ", d.ID, d.Title)
					pending = append(pending, d)
					seen[d.ID] = true
				}
			}
			log.Debug("Feed provider asks for refresh time: ", f.fetcher.Refresh, " will wait ", delay, " before fetching again")
		case errc := <-f.closing:
			errc <- f.err
			close(f.pipe)
			return
		}
	}
}
