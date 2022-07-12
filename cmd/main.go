package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	baseUrl = "https://gmail.googleapis.com/gmail"
)

var (
	Client    *http.Client
	gtFromRex = regexp.MustCompile("(?m)^(>From )")
	fromRex   = regexp.MustCompile("(?m)^(From )")
)

func LoadConfig(file string) (*oauth2.Config, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	var config map[string]interface{}
	err = json.Unmarshal(b, &config)
	if err != nil {
		return nil, err
	}
	nested := config["installed"]
	if nested != nil {
		config = nested.(map[string]interface{})
	}
	return &oauth2.Config{
		ClientID:     config["client_id"].(string),
		ClientSecret: config["client_secret"].(string),
		Endpoint:     google.Endpoint}, nil
}

func Login(config *oauth2.Config) (*oauth2.Token, error) {
	config.Scopes = []string{"https://mail.google.com/"}
	config.RedirectURL = "http://localhost:9191"
	cmd := exec.Command("open", config.AuthCodeURL("state"))
	cmd.Start()
	done := make(chan *oauth2.Token, 1)
	var h http.HandlerFunc
	h = func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Authentication Completed. You can close this page")
		code := r.URL.Query().Get("code")
		t, err := config.Exchange(oauth2.NoContext, code)
		if err != nil {
			log.Fatalf("\nFailed to get access token:%s", err)
			return
		}
		done <- t
	}
	srv := &http.Server{Addr: ":9191", Handler: h}
	go func() {
		srv.ListenAndServe()
	}()
	token := <-done
	srv.Shutdown(context.Background())
	return token, nil
}

func GetAccessToken(file string) (*oauth2.Token, error) {
	b, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var token = oauth2.Token{}
	json.Unmarshal(b, &token)
	return &token, nil
}

type Message struct {
	Id           string
	InternalDate string
	ThreadId     string
	Snippet      string
	SizeEstimate int
	Raw          string
}

// retrieve email ids before given year
func MessagesList(query string, paginate bool) (<-chan Message, error) {
	ch := make(chan Message)
	go func() {
		pageToken := ""
		body := struct {
			Messages           []Message
			NextPageToken      string
			ResultSizeEstimate int
		}{}
		for {
			queryParams := url.Values{"q": []string{query},
				"pageToken": []string{pageToken}}
			r, err := Client.Get(fmt.Sprintf("%s/v1/users/me/messages?%s", baseUrl, queryParams.Encode()))
			if err != nil {
				log.Fatalf("\nFailed listing messages:%s", err)
				break
			}
			b, err := io.ReadAll(r.Body)
			if err != nil {
				log.Fatalf("\nFailed loading messages:%s", err)
				break
			}
			err = json.Unmarshal(b, &body)
			if err != nil {
				log.Fatalf("\nFailed parsing messages:%s, body:%s", err, b)
				break
			}
			for _, m := range body.Messages {
				ch <- m
			}
			if body.NextPageToken == "" || !paginate || len(body.Messages) <= 0 || pageToken == body.NextPageToken {
				break
			}
			pageToken = body.NextPageToken
		}
		close(ch)
	}()
	return ch, nil
}

func MessagesGet(id string) (*Message, error) {
	r, err := Client.Get(fmt.Sprintf("%s/v1/users/me/messages/%s?format=raw", baseUrl, id))
	if err != nil {
		return nil, err
	}
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	var message = Message{}
	err = json.Unmarshal(b, &message)
	if err != nil {
		return nil, err
	}
	return &message, nil
}

func MessagesBatchDelete(ids []string) {
	b, err := json.Marshal(struct {
		Ids []string `json:"ids"`
	}{Ids: ids})
	if err != nil {
		log.Fatalf("\nFailed marshalling message ids: %s", err)
	}
	r, err := Client.Post(fmt.Sprintf("%s/v1/users/me/messages/batchDelete", baseUrl),
		"application/json", bytes.NewReader(b))
	if err != nil {
		log.Fatalf("\nFailed deleting messages: %s", err)
	}
	if r.StatusCode >= 300 {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			log.Fatalf("\nFailed reading API response:%s", err)
		}
		log.Printf("\nFailed deleting messages: %d, response: %s", r.StatusCode, string(b))
	}
}

func ConfigClient() {
	home_dir := os.Getenv("HOME")
	config, err := LoadConfig(home_dir + "/.config/gmail-secret.json")
	if err != nil {
		log.Fatalf("\nFailed loading Gmail client config: %s", err)
	}
	token_file := home_dir + "/.config/gmail-token.json"
	token, err := GetAccessToken(token_file)
	if err != nil || !token.Valid() {
		token, err = Login(config)
		if err != nil {
			log.Fatalf("\nFailed login: %s", err)
		}
		b, err := json.Marshal(token)
		if err != nil {
			log.Fatalf("failed to marshall: %s", err)
		}
		os.WriteFile(token_file, b, 0600)
	}
	Client = config.Client(oauth2.NoContext, token)
}

func DoExport(query string, outFile string, allPages bool, concurMsgGet int) {
	log.Printf("Exporting messages matching: %s to file: %s", query, outFile)
	mmCh, err := MessagesList(query, allPages)
	if err != nil {
		log.Fatalf("\nFailed listing messages: %s", err)
	}
	msgCh := make(chan string)
	wg := sync.WaitGroup{}
	for i := 0; i < concurMsgGet; i++ {
		wg.Add(1)
		go func() {
			for {
				m, ok := <-mmCh
				if ok {
					m, err := MessagesGet(m.Id)
					if err != nil {
						log.Fatalf("\nFailed to retrieve message: %s", err)
					}
					msgCh <- m.Raw
				} else {
					wg.Done()
					break
				}
			}
		}()
	}
	go func() {
		wg.Wait()
		close(msgCh)
	}()
	f, err := os.Create(outFile)
	if err != nil {
		log.Fatalf("\nFailed opening file (%s) :%s", outFile, err)
	}
	defer f.Close()
	msgCount := 0
	messageHeader := fmt.Sprintf("From - %s\n", time.Now().Format("Mon Jan 2 15:04:05 2006"))
	for m := range msgCh {
		msgCount++
		f.WriteString(messageHeader)
		b, err := base64.URLEncoding.DecodeString(m)
		if err != nil {
			log.Fatalf("Failed to decode message: %s", err)
		}
		b = []byte(Escaped(string(b)))
		_, err = f.Write(b)
		if err != nil {
			log.Fatalf("\nFailed to store message: %s", err)
		}
		f.WriteString("\n\n")
		fmt.Printf("\rMessages exported: %d", msgCount)
	}
	fmt.Printf("\rMessages exported: %d", msgCount)
	log.Println("\nDone")
}

func Escaped(m string) string {
	m = gtFromRex.ReplaceAllString(m, ">$1")
	m = fromRex.ReplaceAllString(m, ">$1")
	return m
}

func DoPurge(query string, allPages bool) {
	log.Printf("Deleting messages matching: %s\n", query)
	deleteCount := 0
	for {
		msgsCh, err := MessagesList(query, false)
		if err != nil {
			log.Fatalf("\nFailed to get msg list :%s", err)
		}
		ids := []string{}
		for m := range msgsCh {
			ids = append(ids, m.Id)
			deleteCount++
		}
		if len(ids) > 0 {
			MessagesBatchDelete(ids)
		}
		fmt.Printf("\rMessages deleted:%d", deleteCount)
		if !allPages || len(ids) <= 0 {
			break
		}
	}
	log.Println("\nDone")
}

func main() {
	log.SetFlags(0)
	var query string
	var allPages bool
	var exportCmd = flag.NewFlagSet("export", flag.ExitOnError)
	var purgeCmd = flag.NewFlagSet("purge", flag.ExitOnError)
	var outFile = exportCmd.String("o", "messages.mbox", "Output file to export messages to")
	var concurMsgGet = exportCmd.Int("c", 2, "Number of concurrent message retrievers")
	exportCmd.StringVar(&query, "q", "", "Filter query messages")
	exportCmd.BoolVar(&allPages, "a", false, "Export all or only the first 100 matching messages")
	purgeCmd.StringVar(&query, "q", "", "Filter query messages")
	purgeCmd.BoolVar(&allPages, "a", false, "Purge all(or only the first 100) matching messages")
	usageHelp := func() {
		log.Printf("Usage:\n\t%s <command> [arguments]\n\nThe commands are:\n\n", os.Args[0])
		for _, c := range []struct {
			fs   *flag.FlagSet
			desc string
		}{{fs: exportCmd, desc: "Exports messages to a file"},
			{fs: purgeCmd, desc: "Deletes messages from Gmail"}} {
			fmt.Printf("\t%s\t%s\n", c.fs.Name(), c.desc)
		}
	}
	switch os.Args[1] {
	case exportCmd.Name():
		exportCmd.Parse(os.Args[2:])
		ConfigClient()
		DoExport(query, *outFile, allPages, *concurMsgGet)
	case purgeCmd.Name():
		purgeCmd.Parse(os.Args[2:])
		ConfigClient()
		DoPurge(query, allPages)
	case "-h":
		usageHelp()
	default:
		log.Printf("Invalid sub-command %s\n", os.Args[1])
		usageHelp()
	}
}
