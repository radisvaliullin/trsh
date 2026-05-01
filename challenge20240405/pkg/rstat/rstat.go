package rstat

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

func Main() {

	// config
	conf, err := GetConfig()
	if err != nil {
		log.Fatalf("rstat: get config error, %v", err)
	}

	// init dependencies
	cln := NewClient(conf.Client)
	clnLimiter := NewClientLimiter(cln)

	// init and run rstat
	rstat := New(conf, clnLimiter)
	if err = rstat.Run(); err != nil {
		log.Printf("rstat: run error, %v", err)
	}
}

type RStat struct {
	config Config

	cln IClient

	// just for testing
	// atomic
	numSimultFetch int64

	// time since app track posts (not inclusive)
	// not thread-safe
	firstPostStart float64

	postsMx sync.Mutex
	posts   PostsStat
}

func New(config Config, cln IClient) *RStat {
	s := &RStat{
		config: config,
		cln:    cln,
		posts:  NewPostsStat(),
	}
	return s
}

func (s *RStat) Run() error {
	fmt.Printf("subreddit %s, new posts:\n", s.config.Client.Subreddit)

	// init time since app track posts
	res, err := s.cln.SubredditNew("")
	if err != nil {
		log.Printf("rstat: request error, %v", err)
		return err
	}
	if len(res.Payload.PostsData.Posts) > 0 {
		s.firstPostStart = res.Payload.PostsData.Posts[0].Data.Created
	}

	// set priority channel for handle subfetch prior new fetch
	nextSubFetch := make(chan SubFetch, 1000)
	// stat print ticker
	statTick := time.NewTicker(time.Second * 5)

	// log stat
	go func() {
		for range statTick.C {
			s.printStat()
		}
	}()

	// fetch posts
	for {
		select {
		default:
			// fetch posts (blocking)
			s.fetchPosts(s.firstPostStart, "", nextSubFetch)
		case subFetch := <-nextSubFetch:
			go s.fetchPosts(s.firstPostStart, subFetch.After, nextSubFetch)
		}
	}
}

// fetch posts until end time (not inclusive)
// after used for requst next page
// blocking by client requests limit
func (s *RStat) fetchPosts(end float64, after string, subFetchChan chan SubFetch) {
	atomic.AddInt64(&s.numSimultFetch, 1)
	defer func() {
		atomic.AddInt64(&s.numSimultFetch, -1)
	}()

	res, err := s.cln.SubredditNew(after)
	if err != nil {
		log.Printf("rstat: fetch posts error, %v", err)
		return
	}

	// no data
	if len(res.Payload.PostsData.Posts) == 0 {
		return
	}

	// handle posts
	for idx, post := range res.Payload.PostsData.Posts {
		// handle until end
		if post.Data.Created <= end {
			// update posts before return
			s.updatePosts(res.Payload.PostsData.Posts[:idx])
			// update stat when reach last post
			s.updatePostsStat()
			return
		}
		// print for debug
		// fmt.Printf("new post: %+v\n", post.Data)
	}
	// update posts before return
	s.updatePosts(res.Payload.PostsData.Posts)

	// if need request next page
	after = res.Payload.PostsData.After
	if len(after) == 0 {
		// no next page
		// update stat when reach last post
		s.updatePostsStat()
		return
	}

	// repeat fetch until end
	subFetchChan <- SubFetch{After: after}
}

func (s *RStat) updatePosts(posts []PostData) {
	s.postsMx.Lock()
	defer s.postsMx.Unlock()
	for _, postData := range posts {
		s.posts.Posts[postData.Data.Name] = postData.Data
	}
	// for debug
	// fmt.Printf("update posts: %+v\n", posts)
}

func (s *RStat) updatePostsStat() {
	s.postsMx.Lock()
	defer s.postsMx.Unlock()
	// reset
	s.posts.MostVotedPost = Post{}
	s.posts.UserWithMostPosts = ""
	s.posts.usersPostsCnt = map[string]int{}
	//
	for _, post := range s.posts.Posts {
		if post.Ups > s.posts.MostVotedPost.Ups {
			s.posts.MostVotedPost = post
		}
		//
		s.posts.usersPostsCnt[post.Author]++
		if s.posts.usersPostsCnt[post.Author] > s.posts.usersPostsCnt[s.posts.UserWithMostPosts] {
			s.posts.UserWithMostPosts = post.Author
		}
	}
	// for debug
	// fmt.Printf("update posts stat: %+v\n", s.posts)
}

func (s *RStat) printStat() {
	fmt.Println("Stat:")
	spaces := "        "
	fmt.Printf("%stotal requests - %v\n", spaces, s.cln.GetTotalReqCnt())
	fmt.Printf("%snum simult - %v\n", spaces, atomic.LoadInt64(&s.numSimultFetch))
	s.postsMx.Lock()
	mostVoted := s.posts.MostVotedPost
	maxPostUser := s.posts.UserWithMostPosts
	s.postsMx.Unlock()
	fmt.Printf("%smost voted post - %+v\n", spaces, mostVoted)
	fmt.Printf("%suser with most posts - %v\n", spaces, maxPostUser)
}
