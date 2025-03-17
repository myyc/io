package main

import (
	"encoding/xml"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/parser"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"
)

// Post struct to hold the post data
type Post struct {
	Filename string
	Title    string `yaml:"title"`
	Date     string `yaml:"date"`
	Tags     string `yaml:"tags"`
	Draft    bool   `yaml:"draft"`
	Body     template.HTML
}

// RSS represents the RSS feed
type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel Channel  `xml:"channel"`
}

// Channel represents the RSS channel
type Channel struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Language    string `xml:"language"`
	Items       []Item `xml:"item"`
}

// Item represents an item in the RSS feed
type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
}

// FormatDate converts a date string in RFC3339 format to a formatted date string
func FormatDate(format string, dateStr string) string {
	// Parse the date string in RFC3339 format
	t, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		log.Printf("Error parsing date: %v", err)
		return ""
	}

	// Format the time.Time object according to the provided format
	return t.Format(format)
}

// Trivia returns a random sentence from a list of trivia
func Trivia() string {
	trivia := []string{
		"Your beloved ones love you",
		"Your beloved ones don't love you",
		"You will feel more intelligent",
		"You will feel less intelligent",
		"There is a heaven and you're not going",
		"There is a heaven and you're going",
		"There is no heaven but you're not going anyway",
		"There is no heaven but you're going somewhere else",
		"Your path to enlightenment is blocked by a cat",
		"You will get arrested",
		"Your loneliness will be cured",
		"Your loneliness will be eternal",
		"Your loneliness will be cured by a cat",
		"Your loneliness will be eternal because of a cat",
		"You will be reincarnated as a cat",
		"You will be reincarnated as a cat and you will be lonely",
		"You will be reincarnated as a cat and you will be loved",
		"You will be reincarnated as a cat and you will be loved by a lonely person",
		"You will be reincarnated as a cat and you will be loved by a lonely person who will be arrested",
		"You will be reincarnated as a tree and you will live three thousand years",
		"You will be reincarnated as a tree and you will be cut down",
		"You will be reincarnated as a dog and someone will eat you",
		"You will be reincarnated as a dog and you will eat someone",
		"You will be reincarnated as a sea urchin",
		"You will be reincarnated as a bacterium inside your own body",
		"You will be reincarnated as a maggot who will eat your decaying body",
	}

	// Return a random trivia
	return trivia[time.Now().UnixNano()%int64(len(trivia))]
}

// Create a new template.FuncMap and add the FormatDate function
var funcMap = template.FuncMap{
	"FormatDate": FormatDate,
	"Trivia":     Trivia,
}

// RSSHandler generates the RSS feed
func RSSHandler(w http.ResponseWriter, r *http.Request) {
	posts, err := GetAllPosts()
	if err != nil {
		log.Printf("Error getting all posts: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Filter out drafts
	var rssItems []Item
	for _, post := range posts {
		if !post.Draft {
			// Extract the first two paragraphs
			paragraphs := strings.Split(string(post.Body), "</p>")
			description := ""
			for i, paragraph := range paragraphs {
				if i < 2 {
					description += paragraph + "</p>"
				}
			}

			rssItems = append(rssItems, Item{
				Title:       post.Title,
				Link:        fmt.Sprintf("http://%s/post/%s", r.Host, post.Filename),
				Description: description,
				PubDate:     FormatDate(time.RFC1123, post.Date),
				GUID:        post.Filename,
			})
		}
	}

	rssFeed := RSS{
		Version: "2.0",
		Channel: Channel{
			Title:       "io.",
			Link:        "http://io.myyc.dev",
			Description: "io.myyc.dev",
			Language:    "en-gb",
			Items:       rssItems,
		},
	}

	w.Header().Set("Content-Type", "text/xml")
	w.Header().Set("Content-Disposition", "inline")
	if err := xml.NewEncoder(w).Encode(rssFeed); err != nil {
		log.Printf("Error encoding RSS feed: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Updated GetAllPosts function
func GetAllPosts() ([]Post, error) {
	var posts []Post
	files, err := filepath.Glob("posts/*.md")
	if err != nil {
		log.Printf("Error finding posts: %v", err)
		return nil, err
	}

	for _, file := range files {
		log.Printf("Reading file: %s", file)
		post, err := parsePost(file)
		if err != nil {
			log.Printf("Error parsing post %s: %v", file, err)
			continue
		}

		post.Filename = filepath.Base(file)
		posts = append(posts, post)
	}

	// Sort posts by date in descending order
	sort.Slice(posts, func(i, j int) bool {
		return strings.Compare(posts[i].Date, posts[j].Date) > 0
	})

	log.Printf("Total posts found: %d", len(posts))
	return posts, nil
}

// GetPost retrieves a single post by filename
func GetPost(filename string) (Post, error) {
	file := filepath.Join("posts", filepath.Clean(filename))

	if !strings.HasPrefix(file, "posts") {
		return Post{}, os.ErrNotExist
	}

	if _, err := os.Stat(file); os.IsNotExist(err) {
		return Post{}, os.ErrNotExist
	}

	// Assuming parsePost reads the file and parses it into a Post struct
	return parsePost(file)
}

// parsePost reads a Markdown file, parses its YAML front matter and Markdown content, then returns a Post struct
func parsePost(filename string) (Post, error) {
	var post Post = Post{
		Draft: false,
	}

	// Read the Markdown file content
	content, err := os.ReadFile(filename)
	if err != nil {
		log.Printf("Error reading file %s: %v", filename, err)
		return post, err
	}

	// Split the content into YAML front matter and Markdown body
	parts := strings.SplitN(string(content), "\n---\n", 2)
	if len(parts) < 2 {
		log.Printf("Error: File %s does not contain valid front matter", filename)
		return post, fmt.Errorf("invalid front matter")
	}

	// Parse the YAML front matter
	err = yaml.Unmarshal([]byte(parts[0]), &post)
	if err != nil {
		log.Printf("Error parsing YAML in file %s: %v", filename, err)
		return post, err
	}

	// Setup the Markdown parser with footnote extension
	extensions := parser.CommonExtensions | parser.Footnotes
	mdParser := parser.NewWithExtensions(extensions)

	// Convert Markdown to HTML with footnote support
	html := markdown.ToHTML([]byte(parts[1]), mdParser, nil)
	post.Body = template.HTML(html)

	return post, nil
}

// IndexHandler handles the index page
func IndexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.New("layout.html").Funcs(funcMap).ParseFiles("templates/layout.html", "templates/index.html")
	if err != nil {
		log.Printf("Error parsing templates: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	posts, err := GetAllPosts()
	if err != nil {
		log.Printf("Error getting all posts: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		IsHome bool
		Posts  []Post
	}{
		IsHome: true,
		Posts:  posts,
	}

	if err := tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// PostHandler handles individual post pages
func PostHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	title := vars["title"]
	tmpl, err := template.New("layout.html").Funcs(funcMap).ParseFiles("templates/layout.html", "templates/post.html")
	if err != nil {
		log.Printf("Error parsing templates: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	post, err := GetPost(title)
	if os.IsNotExist(err) {
		log.Printf("Post not found: %s", title)
		http.NotFound(w, r)
		return
	} else if err != nil {
		log.Printf("Error getting post: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := struct {
		IsHome bool
		Post   Post
	}{
		IsHome: false,
		Post:   post,
	}

	if err := tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/", IndexHandler).Methods("GET")
	r.HandleFunc("/post/{title}", PostHandler).Methods("GET")
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))
	r.HandleFunc("/feed.xml", RSSHandler).Methods("GET") // Add this line

	log.Println("Starting server on :8081")
	if err := http.ListenAndServe(":8081", r); err != nil {
		log.Fatalf("could not start server: %s\n", err)
	}
}
