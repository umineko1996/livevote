package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	youtube "google.golang.org/api/youtube/v3"
)

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
	}
}

type Choice struct {
	number string
	value  string
}

func (c Choice) String() string {
	return c.number + ".\"" + c.value + "\""
}

func NewChoices(choices []string) []Choice {
	cs := []Choice{}
	for i, c := range choices {
		_c := Choice{number: strconv.Itoa(i + 1), value: c}
		cs = append(cs, _c)
	}
	return cs
}

var voter = make(map[string]bool)
var votes = make(map[string]int)
var choices = NewChoices([]string{"1", "2"})

func MessageHandle(message *youtube.LiveChatMessage) error {
	userID, text := message.AuthorDetails.ChannelId, message.Snippet.TextMessageDetails.MessageText

	if voter[userID] {
		// 投票済み
		return nil
	}
	for _, c := range choices {
		// 選択肢の単語を含んでいれば投票数をプラス
		if strings.Contains(text, c.number) || strings.Contains(text, c.value) {
			votes[c.value]++
			voter[userID] = true
			fmt.Printf("vote %s %s(&%s) text:\"%s\"\n", c, message.AuthorDetails.DisplayName, userID, text)
			break
		}
	}
	return nil
}

func run() error {
	videoID, voteTime, startTime := getArgs()
	if videoID == "" || voteTime == 0 {
		return fmt.Errorf("usage: %s -id video_id [-t voting_time] [-s start_wait_time] [choice1 choice2, ...]", path.Base(os.Args[0]))
	}

	client, err := NewClient()
	if err != nil {
		return err
	}

	service, err := youtube.New(client)
	if err != nil {
		return fmt.Errorf("new client: %w", err)
	}

	getLiveID := func(videoID string) (liveChatID string, err error) {
		call := service.Videos.List("liveStreamingDetails").Id(videoID)
		resp, err := call.Do()
		if err != nil {
			return "", fmt.Errorf("get broadcast: %w", err)
		} else if len(resp.Items) == 0 {
			// b, _ := resp.MarshalJSON()
			// fmt.Println(string(b))
			return "", errors.New("get broadcast: Not Found")
		}
		return resp.Items[0].LiveStreamingDetails.ActiveLiveChatId, nil
	}

	chatID, err := getLiveID(videoID)
	if err != nil {
		return err
	}

	fmt.Printf("start voting in %ds seconds...\n", startTime)
	fmt.Println("if you stop vote halfway, please press enter key")
	for _, c := range choices {
		fmt.Println(c)
	}
	fmt.Println("")
	fmt.Println("vote start!")
	fmt.Println("--------------------")

	// 投票開始用のページ取得
	resp, err := service.LiveChatMessages.List(chatID, "id").Do()
	if err != nil {
		return fmt.Errorf("get livechat: %w", err)
	}
	time.Sleep(time.Duration(resp.PollingIntervalMillis) * time.Millisecond)

	call := service.LiveChatMessages.List(chatID, "snippet, AuthorDetails")
	next := resp.NextPageToken
	delay := 5
	loop := voteTime / delay
	timer := time.NewTimer(0)
	defer timer.Stop()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	input := makeInputWaitCh(ctx)

	if (voteTime % delay) != 0 {
		loop++
	}
	for i := 0; i < loop; i++ {
		delay = 5
		delayMs := int64(delay * 1000)
		if resp.PollingIntervalMillis > int64(delay)*1000 {
			delayMs = resp.PollingIntervalMillis
			delay = int(delayMs / 1000)
		}
		timer.Reset(time.Duration(delayMs) * time.Millisecond)

		fmt.Printf("time remaining %d sec\n", (loop-i)*delay)
		fmt.Printf("delay %d sec...\n", delay)
		select {
		case <-input:
			i = loop
			cancel()
			time.Sleep(time.Duration(resp.PollingIntervalMillis) * time.Millisecond)
		case <-timer.C:
		}

		// コメント取得
		resp, err := call.PageToken(next).MaxResults(2000).Do()
		if err != nil {
			return fmt.Errorf("get livechat: %w", err)
		}

		for _, item := range resp.Items {
			MessageHandle(item)
		}

		next = resp.NextPageToken
	}
	fmt.Println("--------------------")

	max := 0
	maxChoice := []Choice{}

	fmt.Println("vote end!")
	if ctx.Err() != nil {
		fmt.Println("vote ended press enter key")
	}
	fmt.Println("")

	fmt.Printf("total vote %d\n", len(voter))
	fmt.Println("--------------------")
	for _, c := range choices {
		vote := votes[c.value]
		if vote > max {
			max = vote
			maxChoice = []Choice{c}
		} else if vote == max {
			maxChoice = append(maxChoice, c)
		}
		fmt.Printf("%s vote %d\n", c, vote)
	}
	fmt.Println("--------------------")
	fmt.Printf("Winning vote %d\n", max)
	for _, mc := range maxChoice {
		fmt.Println(mc)
	}
	fmt.Println()

	return nil
}

func makeInputWaitCh(ctx context.Context) <-chan string {
	ch := make(chan string)

	go func() {
		defer close(ch)
		b := make([]byte, 1)
		for {
			select {
			case <-ctx.Done():
				break
			default:
			}
			// select待機だと待機中に入力受け付けられないっぽいのでスリープを使う
			time.Sleep(100 * time.Millisecond)
			os.Stdin.Read(b)
			ch <- string(b)
		}
	}()

	return ch
}

func getArgs() (videoID string, voteTime, startTime int) {
	flag.StringVar(&videoID, "id", "", "youtube video id")
	flag.IntVar(&voteTime, "t", 30, "voting time. default 30sec")
	flag.IntVar(&startTime, "s", 3, "wait time before start voting. default 3sec")
	flag.Parse()

	if flag.NArg() != 0 {
		choices = NewChoices(flag.Args())
	}

	return videoID, voteTime, startTime
}
