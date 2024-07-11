package main

import (
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

// Create a new template.FuncMap and add the FormatDate function
var funcMap = template.FuncMap{
	"FormatDate": FormatDate,
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

	if err := tmpl.ExecuteTemplate(w, "layout.html", posts); err != nil {
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

	if err := tmpl.ExecuteTemplate(w, "layout.html", post); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/", IndexHandler).Methods("GET")
	r.HandleFunc("/post/{title}", PostHandler).Methods("GET")
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))

	log.Println("Starting server on :8081")
	if err := http.ListenAndServe(":8081", r); err != nil {
		log.Fatalf("could not start server: %s\n", err)
	}
}
