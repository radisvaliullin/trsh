package rstat

type Resp struct {
	Header  Header
	Payload Payload
}

type Header struct {
	Used      int
	Remaining int
	Reset     int
}

type Payload struct {
	PostsData PostsData `json:"data"`
}

type PostsData struct {
	After  string     `json:"after"`
	Dist   int        `json:"dist"`
	Posts  []PostData `json:"children"`
	Before string     `json:"before"`
}

type PostData struct {
	Data Post `json:"data"`
}

type Post struct {
	Title   string  `json:"title"`
	Name    string  `json:"name"`
	Ups     int     `json:"ups"`
	Author  string  `json:"author"`
	Created float64 `json:"created"`
}

type PostsStat struct {
	Posts             map[string]Post
	MostVotedPost     Post
	UserWithMostPosts string
	usersPostsCnt     map[string]int
}

func NewPostsStat() PostsStat {
	ps := PostsStat{
		Posts:         map[string]Post{},
		usersPostsCnt: map[string]int{},
	}
	return ps
}

type SubFetch struct {
	After string
}
