package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	youtube "google.golang.org/api/youtube/v3"
)

var localhost = "localhost:8090"
var cacheFile = "client_code"

func getTokenFromFile(file string) (*oauth2.Token, error) {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	token := new(oauth2.Token)
	if err := json.Unmarshal(b, token); err != nil {
		log.Println(err)
		return nil, err
	}

	return token, nil
}

func existsCode(code string) bool {
	return code != ""
}

func getToken(config *oauth2.Config) (*oauth2.Token, error) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)

	if token, err := getTokenFromFile(cacheFile); err == nil {
		return token, nil
	}

	token, err := getTokenFromWeb(config, authURL)
	if err != nil {
		return nil, err
	}

	// 失敗しても気にしない
	func() {
		cache, err := json.Marshal(token)
		if err != nil {
			log.Println(err)
			return
		}
		f, err := os.Create(cacheFile)
		if err != nil {
			log.Println(err)
			return
		}
		defer f.Close()
		f.Write([]byte(cache))
	}()

	return token, nil
}

func NewClient() (*http.Client, error) {
	config := &oauth2.Config{
		RedirectURL:  "http://" + localhost,
		ClientID:     "XXXXXXXXXXXXXXXXXX.apps.googleusercontent.com",
		ClientSecret: "XXXXXXXXXXXXXXXXXX",
		Scopes:       []string{"email", youtube.YoutubeReadonlyScope},
		Endpoint:     google.Endpoint,
	}

	token, err := getToken(config)
	if err != nil {
		return nil, err
	}

	return config.Client(context.Background(), token), nil
}

// getTokenFromWeb uses Config to request a Token.
// It returns the retrieved Token.
func getTokenFromWeb(config *oauth2.Config, authURL string) (*oauth2.Token, error) {
	codeCh, err := startWebServer()
	if err != nil {
		fmt.Printf("Unable to start a web server.")
		return nil, err
	}

	err = openURL(authURL)
	if err != nil {
		log.Fatalf("Unable to open authorization URL in web server: %v", err)
	} else {
		fmt.Println("Your browser has been opened to an authorization URL.",
			" This program will resume once authorization has been provided.")
		fmt.Println(authURL)
	}

	// Wait for the web server to get the code.
	code := <-codeCh

	fmt.Println(code)

	return exchangeToken(config, code)
}

// startWebServer starts a web server that listens on http://localhost:8080.
// The webserver waits for an oauth code in the three-legged auth flow.
func startWebServer() (codeCh chan string, err error) {
	listener, err := net.Listen("tcp", localhost)
	if err != nil {
		return nil, err
	}
	codeCh = make(chan string)

	go http.Serve(listener, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		code := r.FormValue("code")
		codeCh <- code // send code to OAuth flow
		listener.Close()
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "Received code: %v\r\nYou can now safely close this browser window.", code)
	}))

	return codeCh, nil
}

// openURL opens a browser window to the specified location.
// This code originally appeared at:
//   http://stackoverflow.com/questions/10377243/how-can-i-launch-a-process-that-is-not-a-file-in-go
func openURL(url string) error {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("Cannot open URL %s on this platform", url)
	}
	return err
}

// Exchange the authorization code for an access token
func exchangeToken(config *oauth2.Config, code string) (*oauth2.Token, error) {
	tok, err := config.Exchange(oauth2.NoContext, code)
	if err != nil {
		log.Fatalf("Unable to retrieve token %v", err)
	}
	return tok, nil
}
