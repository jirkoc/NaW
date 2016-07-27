package handlers

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
)

// Page holds the structure for a templated page.
type Page struct {
	Title       string
	Body        []byte
	Stamp       string
	Size        string
	DisplayBody template.HTML
}

// save saves Pages.
func (p *Page) save() error {
	filename := "data/" + p.Title + ".txt"
	return ioutil.WriteFile(filename, p.Body, 0600)
}

// loadPage loads Pages.
func loadPage(title string) (*Page, error) {
	filename := title + ".txt"
	body, err := ioutil.ReadFile("data/" + filename)
	if err != nil {
		return nil, err
	}
	fileInfo, err := os.Stat("data/" + filename)
	if err != nil {
		return nil, err
	}
	stamp := fileInfo.ModTime().Format("2. Jan 2006, 15:04:05")
	var size string
	sizeRaw := fileInfo.Size()
	if sizeRaw >= 1024 {
		size = fmt.Sprintln((sizeRaw / 1024), "KiB")
	} else {
		size = fmt.Sprintln(sizeRaw, "Bytes")
	}

	return &Page{Title: title, Body: body, Stamp: stamp, Size: size}, nil
}

// Template caching to reduce calling of ParseFiles.
var templates = template.Must(template.ParseFiles("tmpl/edit.html", "tmpl/view.html"))

// A renderTemplate function to avoid code duplication.
func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates.ExecuteTemplate(w, tmpl+".html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// This expression helps converting Page names into links.
var pageRegexp = regexp.MustCompile("\\[([0-9A-Za-z]+)\\]")

// This expression helps creating headlines.
// TODO Find a better expression for alphanumeric characters including german Umlauts.
var headFirRegexp = regexp.MustCompile("#\\s([0-9A-Za-zÄäÖöÜüß./_:@\\- ]+)")
var headSecRegexp = regexp.MustCompile("#{2}\\s([0-9A-Za-zÄäÖöÜüß./_:@\\- ]+)")
var headThiRegexp = regexp.MustCompile("#{3}\\s([0-9A-Za-zÄäÖöÜüß./_:@\\- ]+)")

// This expression helps creating visual line breaks.
var breakRegexp = regexp.MustCompile("\n")

// These expressions help creating clickable hyperlinks.
var sLinkRegexp = regexp.MustCompile("\\[\\[([0-9A-Za-z./_:@\\-]+)\\]\\]")
var nLinkRegexp = regexp.MustCompile("\\[\\[([0-9A-Za-z./_:@\\-]+)\\|([0-9A-Za-zÄäÖöÜüß!-~ ]+)\\]\\]")

// prepBody prepares the Body of the Page for the View function.
// TODO Implement more MarkDown syntax (what about an MarkDown library?)
func prepBody(escBody []byte) []byte {
	// TODO This looks pretty ugly but I'm too tired now. In other words:
	// FIXME This section needs some cleanup/code deduplication.
	// Wrapper function which accepts a compiled regular expression plus the
	// "out string"?
	preppedBody := pageRegexp.ReplaceAllFunc(escBody, func(str []byte) []byte {
		matched := pageRegexp.FindStringSubmatch(string(str))
		out := []byte("<a href=\"/view/" + matched[1] + "\">" + matched[1] + "</a>")
		return out
	})
	preppedBody = headThiRegexp.ReplaceAllFunc(preppedBody, func(str []byte) []byte {
		matched := headThiRegexp.FindStringSubmatch(string(str))
		out := []byte("<h3>" + matched[1] + "</h3>")
		return out
	})
	preppedBody = headSecRegexp.ReplaceAllFunc(preppedBody, func(str []byte) []byte {
		matched := headSecRegexp.FindStringSubmatch(string(str))
		out := []byte("<h2>" + matched[1] + "</h2>")
		return out
	})
	preppedBody = headFirRegexp.ReplaceAllFunc(preppedBody, func(str []byte) []byte {
		matched := headFirRegexp.FindStringSubmatch(string(str))
		out := []byte("<h1>" + matched[1] + "</h1>")
		return out
	})
	preppedBody = breakRegexp.ReplaceAllFunc(preppedBody, func(str []byte) []byte {
		matched := breakRegexp.FindStringSubmatch(string(str))
		out := []byte(matched[0] + "<br/>")
		return out
	})
	preppedBody = sLinkRegexp.ReplaceAllFunc(preppedBody, func(str []byte) []byte {
		matched := sLinkRegexp.FindStringSubmatch(string(str))
		out := []byte("<a href=\"" + matched[1] + "\">" + matched[1] + "</a>")
		return out
	})
	preppedBody = nLinkRegexp.ReplaceAllFunc(preppedBody, func(str []byte) []byte {
		matched := nLinkRegexp.FindStringSubmatch(string(str))
		out := []byte("<a href=\"" + matched[1] + "\">" + matched[2] + "</a>")
		return out
	})

	return preppedBody
}

// View will allow users to view a wiki page.
// It will handle URLs prefixed with "/view/".
func View(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}
	escapedBody := []byte(template.HTMLEscapeString(string(p.Body)))
	preparedBody := prepBody(escapedBody)
	p.DisplayBody = template.HTML(preparedBody)
	renderTemplate(w, "view", p)
}

// Edit loads the page (or, if it doesn't exist, create an empty Page struct),
// and displays an HTML form.
func Edit(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	renderTemplate(w, "edit", p)
}

// Save will handle the submission of forms located on the edit pages.
func Save(w http.ResponseWriter, r *http.Request, title string) {
	body := r.FormValue("body")
	p := &Page{Title: title, Body: []byte(body)}
	err := p.save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

// Root will redirect to the Home of this wiki.
func Root(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/view/Home", http.StatusFound)
}

// As you may have observed, this program has a serious security flaw: a user can
// supply an arbitrary path to be read/written on the server. To mitigate this,
// we can write a function to validate the title with a regular expression:
var validPath = regexp.MustCompile("^/(edit|save|static|view)/([a-zA-Z0-9]+)$")

// Make is a wrapper function that returns a function of type http.HandlerFunc.
func Make(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := validPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		fn(w, r, m[2])
	}
}
