package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/umineko1996/livechathandler"
	youtube "google.golang.org/api/youtube/v3"
)

func main() {
	os.Setenv("GOOGLE_API_CLIENTID", "XXXXXXX.apps.googleusercontent.com")
	os.Setenv("GOOGLE_API_CLIENTSERCRET", "XXXXXXX")

	if err := run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

type arguments struct {
	videoID       string
	votingTime    int
	countdownTime int
	multiple      int
	choice        []string
}

type VoteManager struct {
	VotingTime int
	Multiple   int
	Choices    []Choice
	voter      map[string]int
	votes      map[string]int
	Ctx        context.Context    // ポーリングコンテキスト
	Cancel     context.CancelFunc // ポーリングコンテキストキャンセル関数
	timeout    context.Context    // 投票時間管理コンテキスト
}

type Choice struct {
	number string
	value  string
}

func NewChoices(choices []string) []Choice {
	cs := []Choice{}
	for i, c := range choices {
		_c := Choice{number: strconv.Itoa(i + 1), value: c}
		cs = append(cs, _c)
	}
	return cs
}

func (c Choice) String() string {
	return c.number + ".\"" + c.value + "\""
}

func (vm *VoteManager) MessageHandle(message *youtube.LiveChatMessage) {
	userID, text := message.AuthorDetails.ChannelId, message.Snippet.DisplayMessage
	fmt.Printf("%s: %s\n", message.AuthorDetails.DisplayName, text)
	if vm.voter[userID] >= vm.Multiple {
		// 投票済み
		return
	}
	for _, c := range vm.Choices {
		// 選択肢の単語を含んでいれば投票数をプラス
		if strings.Contains(text, c.number) || strings.Contains(text, c.value) {
			vm.votes[c.value]++
			vm.voter[userID]++
			fmt.Printf("vote %s %s(&%s) text:\"%s\"\n", c, message.AuthorDetails.DisplayName, userID, text)
			break
		}
	}
	return
}

func (vm *VoteManager) IntervalHandle(pollingIntervalMillis int64) {
	if vm.timeout == nil {
		// ハンドラの初回実行時
		fmt.Println("vote start!")
		fmt.Println("--------------------")

		// 投票タイマーをセット
		votingTime := time.Duration(vm.VotingTime) * time.Second
		votingCtx, cancel := context.WithTimeout(vm.Ctx, votingTime)
		go func() {
			defer vm.Cancel()
			defer cancel()
			<-votingCtx.Done()
		}()
		vm.timeout = votingCtx
	}
	t, _ := vm.timeout.Deadline()
	fmt.Printf("remaining time %d sec. delay %d Millis\n",
		int(t.Sub(time.Now()).Seconds()), pollingIntervalMillis)
}

func run() error {
	args := getArgs()
	if err := validateArgs(args); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	voteManager := &VoteManager{
		Ctx:        ctx,
		Cancel:     cancel,
		VotingTime: args.votingTime,
		Multiple:   args.multiple,
		Choices:    NewChoices(args.choice),
		voter:      make(map[string]int),
		votes:      make(map[string]int),
	}

	handler, err := livechathandler.New(args.videoID, livechathandler.WithIntervalHandler(voteManager))
	if err != nil {
		return err
	}

	fmt.Printf("start voting after %ds...\n", args.countdownTime)
	fmt.Println("if you stop vote halfway, please press enter key")
	for _, c := range voteManager.Choices {
		fmt.Println(c)
	}
	for i := args.countdownTime; i > 0; i-- {
		fmt.Printf("%d ", i)
		time.Sleep(1 * time.Second)
	}
	fmt.Println("")
	fmt.Println("")

	// polling end with press enter
	go func() {
		b := make([]byte, 1)
		for {
			select {
			case <-ctx.Done():
				break
			default:
			}
			time.Sleep(100 * time.Millisecond)
			os.Stdin.Read(b)
			if len(b) != 0 {
				cancel()
			}
		}
	}()

	handler.Polling(ctx, voteManager)

	// fmt.Println("--------------------")

	// max := 0
	// maxChoice := []Choice{}

	// fmt.Println("vote end!")
	// if ctx.Err() != nil {
	// 	fmt.Println("vote ended press enter key")
	// }
	// fmt.Println("")

	// fmt.Printf("total vote %d\n", len(voteManager.voter))
	// fmt.Println("--------------------")
	// for _, c := range voteManager.Choices {
	// 	vote := voteManager.voter[c.value]
	// 	if vote > max {
	// 		max = vote
	// 		maxChoice = []Choice{c}
	// 	} else if vote == max {
	// 		maxChoice = append(maxChoice, c)
	// 	}
	// 	fmt.Printf("%s vote %d\n", c, vote)
	// }
	// fmt.Println("--------------------")
	// fmt.Printf("Winning vote %d\n", max)
	// for _, mc := range maxChoice {
	// 	fmt.Println(mc)
	// }
	fmt.Println()

	return nil
}

func getArgs() (args arguments) {
	flag.StringVar(&args.videoID, "id", "", "youtube video id")
	flag.IntVar(&args.votingTime, "t", 30, "voting time. default 30sec")
	flag.IntVar(&args.countdownTime, "s", 3, "wait time before start voting. default 3sec")
	flag.IntVar(&args.multiple, "m", 1, "enable multiple votes. number of votes per a user")
	flag.Parse()

	if flag.NArg() != 0 {
		args.choice = flag.Args()
	} else {
		args.choice = []string{"１", "２"}
	}

	return args
}

var usage = fmt.Sprintf("usage: %s -id video_id [-t voting_time] [-s start_wait_time] [choice1 choice2, ...]", filepath.Base(os.Args[0]))
var ErrValidateFailed = errors.New(usage)

func validateArgs(args arguments) error {
	if args.videoID == "" || args.votingTime == 0 || args.multiple <= 0 {
		return ErrValidateFailed
	}

	return nil
}
