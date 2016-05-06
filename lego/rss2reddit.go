package lego

import (
	"fmt"
	"sync"
	"time"

	"github.com/SlyMarbo/rss"
	"github.com/aggrolite/geddit"
	"github.com/lunny/html2md"
)

// CreateRedditCrossFilter returns a filter function which removes items if they match items from the
// filterSource stream.
func CreateRedditCrossFilter(filterSource Puller, warmupEntries int, warmupDuration time.Duration) func(interface{}) bool {

	filter := struct {
		sync.Mutex
		F map[string]bool
	}{F: make(map[string]bool)}

	// Fill up the filter from filterSource stream.
	// TODO: replace this with a FunctionSink brick after brick.loop() is implemented and
	// supports shutdown on channel close option.
	go func() {
		in := filterSource.Pull()
		for {
			d, ok := <-in
			if !ok {
				return
			}
			title := d.(*geddit.Submission).Title
			filter.Lock()
			filter.F[title] = true
			filter.Unlock()
		}
	}()

	warmTime := time.Now().Add(warmupDuration)
	isWarm := false
	return func(d interface{}) bool {
		title := d.(*geddit.NewSubmission).Title
		for !isWarm && time.Now().Before(warmTime) {
			filter.Lock()
			if len(filter.F) >= warmupEntries {
				isWarm = true
			}
			filter.Unlock()
			// FIXME: ugly spinlocking here. is there a select solution?
			time.Sleep(time.Second)
		}
		filter.Lock()
		found := filter.F[title]
		filter.Unlock()
		return !found
	}
}

// CreateRssToRedditTransform creates a transformer function which converts rss/atom posts to
// reddit submissions.
func CreateRssToRedditTransform(subreddit string) func(interface{}) interface{} {
	return func(in interface{}) interface{} {
		r := in.(*rss.Item)
		content := html2md.Convert(r.Content)
		attribution := fmt.Sprintf("\n[Post](%s) on %s", r.Link, r.Date)
		content += attribution
		return geddit.NewTextSubmission(
			subreddit,
			r.Title,
			content,
			false,
			nil)
	}
}
