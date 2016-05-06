package main

import (
	log "github.com/Sirupsen/logrus"
	"net/http"
	_ "net/http/pprof"
	"time"

	"scribe/lego"

	// "github.com/SlyMarbo/rss"
	// "github.com/aggrolite/geddit"
)

func init() {
	log.SetLevel(log.DebugLevel)
}

/**** Map of blocks ****/
/*
rssFeed ---> FeedToReddit ---> crossFilter ---> redditChannel
                                    ^
                               redditUser
*/
func main() {
	// We should have pprof listening on localhost:6060/debug/pprof
	go func() {
		log.Info("Starting debug http/pprof endpoint listening on http://localhost:6060/debug/pprof")
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	redditKeys := lego.Reddit{
		ClientID:     "",
		ClientSecret: "",
		Username:     "",
		Password:     "",
	}

	// RSS Source
	inF := (&lego.Feed{"http://feeds.feedburner.com/Metafilter"}).Intake()
	defer lego.CloseAndLog(inF)

	// Reddit source to filter already present items.
	inR := (&lego.RedditUser{
		Reddit:  redditKeys,
		User:    "rss_scribe",
		Listing: "submissions"}).Intake()
	defer lego.CloseAndLog(inR)

	// Transform from rss item to reddit submission
	trFtoR := (&lego.Transformer{
		lego.CreateRssToRedditTransform("mefiscribed")}).Transfer()
	defer lego.CloseAndLog(trFtoR)

	filter := (&lego.Sieve{
		lego.CreateRedditCrossFilter(inR, 20, time.Minute)}).Transfer()
	defer lego.CloseAndLog(filter)

	outR := (&lego.RedditSub{
		Reddit:    redditKeys,
		SubReddit: "mefiscribed"}).Output()
	defer lego.CloseAndLog(outR)

	link1 := lego.Chain(inF, trFtoR)
	defer lego.CloseAndLog(link1)
	link2 := lego.Chain(trFtoR, filter)
	defer lego.CloseAndLog(link2)
	link3 := lego.Chain(filter, outR)
	defer lego.CloseAndLog(link3)

	duration := time.Hour
	time.Sleep(duration)
}
