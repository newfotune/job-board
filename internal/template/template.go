package template

import (
	"log"
	"net/http"
	"strings"
	"time"

	stdtemplate "html/template"

	customtemplate "github.com/alecthomas/template"
	humanize "github.com/dustin/go-humanize"
	"github.com/fsnotify/fsnotify"
	blackfriday "gopkg.in/russross/blackfriday.v2"
)

type Template struct {
	templates *customtemplate.Template
	funcMap   stdtemplate.FuncMap
	watcher   *fsnotify.Watcher
}

func NewTemplate(env string) *Template {
	funcMap := customtemplate.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"last": func(a []int) int {
			if len(a) == 0 {
				return -1
			}
			return a[len(a)-1]
		},
		"jsescape":  customtemplate.JSEscapeString,
		"humantime": humanize.Time,
		"humannumber": func(n int) string {
			return humanize.Comma(int64(n))
		},
		"isTimeBeforeNow": func(t time.Time) bool {
			return t.Before(time.Now())
		},
		"isTimeAfterNow": func(t time.Time) bool {
			return t.After(time.Now())
		},
		"truncateName": func(s string) string {
			parts := strings.Split(s, " ")
			return parts[0]
		},
		"stringTitle": func(s string) string {
			return strings.Title(s)
		},
		"replaceDash": func(s string) string {
			return strings.ReplaceAll(s, "-", " ")
		},
		"mul": func(a int, b int) int {
			return a * b
		},
		"currencysymbol": func(currency string) string {
			symbols := map[string]string{
				"USD": "$",
				"EUR": "€",
				"JPY": "¥",
				"GBP": "£",
				"AUD": "A$",
				"CAD": "C$",
				"CHF": "Fr",
				"CNY": "元",
				"HKD": "HK$",
				"NZD": "NZ$",
				"SEK": "kr",
				"KRW": "₩",
				"SGD": "S$",
				"NOK": "kr",
				"MXN": "MX$",
				"INR": "₹",
				"RUB": "₽",
				"ZAR": "R",
				"TRY": "₺",
				"BRL": "R$",
			}
			symbol, ok := symbols[currency]
			if !ok {
				return "$"
			}
			return symbol
		},
	}

	t := &Template{
		templates: createTemplateFromGlob(funcMap, "static/views/*.html"),
		funcMap:   stdtemplate.FuncMap(funcMap),
	}

	if env != "dev" {
		return t
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}
	// Purposefully not closing watcher. We want to watch for the duration of the programs life.

	// Start listening for events.
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Has(fsnotify.Write) {
					log.Printf("modified file %s, reloading templates", event.Name)
					t.templates = createTemplateFromGlob(funcMap, "static/views/*.html")
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error from file watcher:", err)
			}
		}
	}()

	if err = watcher.Add("static/views"); err != nil {
		panic(err)
	}

	t.watcher = watcher
	return t
}

func createTemplateFromGlob(funcMap customtemplate.FuncMap, glob string) *customtemplate.Template {
	return customtemplate.Must(customtemplate.New("stdtmpl").Funcs(funcMap).ParseGlob(glob))
}

func (t *Template) JSEscapeString(s string) string {
	return customtemplate.JSEscapeString(s)
}

func (t *Template) Render(w http.ResponseWriter, status int, name string, data interface{}) error {
	w.WriteHeader(status)
	return t.templates.ExecuteTemplate(w, name, data)
}

func (t *Template) StringToHTML(s string) stdtemplate.HTML {
	return stdtemplate.HTML(s)
}

func (t *Template) MarkdownToHTML(s string) stdtemplate.HTML {
	renderer := blackfriday.NewHTMLRenderer(blackfriday.HTMLRendererParameters{
		Flags: blackfriday.Safelink |
			blackfriday.NofollowLinks |
			blackfriday.NoreferrerLinks |
			blackfriday.HrefTargetBlank,
	})
	return stdtemplate.HTML(blackfriday.Run([]byte(s), blackfriday.WithRenderer(renderer)))
}
