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

func (c Choice) IsSelected(ballot string) bool {
	return strings.Contains(ballot, c.number) || strings.Contains(ballot, c.value)
}

type VoteManager struct {
	votingTime  int
	multiple    int
	countdown   int
	choice      []Choice
	voterList   map[string]int
	votes       map[Choice]int
	totalBallot int
	ctx         context.Context    // ポーリングコンテキスト
	cancel      context.CancelFunc // ポーリングコンテキストキャンセル関数
	timeout     context.Context    // 投票時間管理コンテキスト
}

func NewVoteManager(votingTime, multiple, countdown int, choice []string) *VoteManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &VoteManager{
		ctx:         ctx,
		cancel:      cancel,
		votingTime:  votingTime,
		multiple:    multiple,
		countdown:   countdown,
		totalBallot: 0,
		choice:      NewChoices(choice),
		voterList:   make(map[string]int),
		votes:       make(map[Choice]int),
	}
}

func (vm *VoteManager) Ctx() context.Context {
	return vm.ctx
}

func (vm *VoteManager) isVoting() bool {
	return vm.ctx.Err() == nil
}

func (vm *VoteManager) EndVoting() {
	vm.cancel()
}

func (vm *VoteManager) Vote(c Choice, voter string) {
	vm.votes[c]++
	vm.voterList[voter]++
	vm.totalBallot++
}

func (vm *VoteManager) hasBallot(voter string) bool {
	return vm.voterList[voter] < vm.multiple
}

func (vm *VoteManager) MessageHandle(message *youtube.LiveChatMessage) {
	userID, text := message.AuthorDetails.ChannelId, message.Snippet.DisplayMessage
	// fmt.Printf("%s: %s\n", message.AuthorDetails.DisplayName, text)
	if !vm.hasBallot(userID) {
		// 投票済み
		return
	}
	for _, c := range vm.choice {
		// 選択肢の単語を含んでいれば投票数をプラス
		if c.IsSelected(text) {
			vm.Vote(c, userID)
			fmt.Printf("vote %s %s(&%s) text:\"%s\"\n", c, message.AuthorDetails.DisplayName, userID, text)
			break
		}
	}
	return
}

func (vm *VoteManager) IntervalHandle(pollingIntervalMillis int64) {
	if vm.timeout == nil {
		// ハンドラの初回実行時
		vm.PrintStartMessage()

		// 投票タイマーをセット
		votingTime := time.Duration(vm.votingTime) * time.Second
		votingCtx, cancel := context.WithTimeout(vm.ctx, votingTime)
		go func() {
			defer vm.EndVoting()
			defer cancel()
			<-votingCtx.Done()
		}()
		vm.timeout = votingCtx
	}
	t, _ := vm.timeout.Deadline()
	fmt.Printf("remaining time %d sec. delay %d Millis\n",
		int(t.Sub(time.Now()).Seconds()), pollingIntervalMillis)
}

func (vm *VoteManager) StartInterruptionProcess() {
	go func() {
		b := make([]byte, 1)
		for vm.isVoting() {
			time.Sleep(100 * time.Millisecond)
			os.Stdin.Read(b)
			if len(b) != 0 {
				vm.EndVoting()
			}
		}
	}()
}

func (vm *VoteManager) PrintPreMessage() {
	fmt.Println("*-------------------")
	fmt.Println("|Choice")
	fmt.Println("*-------------------")
	for _, c := range vm.choice {
		fmt.Println("|", c)
	}
	fmt.Println("*-------------------")
	fmt.Println("if you stop vote halfway, please press enter key")
	if vm.countdown > 0 {
		fmt.Printf("start voting after %ds...\n", vm.countdown)
		for i := vm.countdown; i > 0; i-- {
			fmt.Printf("%d ", i)
			time.Sleep(1 * time.Second)
		}
		fmt.Println("")
	}
	fmt.Println("")
}

func (vm *VoteManager) PrintStartMessage() {
	fmt.Println(">------------------<")
	fmt.Println("   start voting!")
	fmt.Println(">------------------<")
}

func (vm *VoteManager) PrintEndMessage() {
	fmt.Println(">------------------<")
	fmt.Println("   end voting!")
	fmt.Println(">------------------<")
	fmt.Println("")
}

func (vm *VoteManager) PrintVoteResultMessage() {
	fmt.Println("*-------------------")
	fmt.Println("|Result")
	fmt.Println("*-------------------")
	max := 0
	maxChoice := []Choice{}
	for _, c := range vm.choice {
		vote := vm.votes[c]
		if vote > max {
			max = vote
			maxChoice = []Choice{c}
		} else if vote == max {
			maxChoice = append(maxChoice, c)
		}
		fmt.Printf("| %s vote %d\n", c, vote)
	}
	fmt.Println("*-------------------")
	fmt.Printf("| total ballot %d\n", vm.totalBallot)
	fmt.Println("*-------------------")
	fmt.Println("|Winning")
	fmt.Println("*-------------------")
	for _, mc := range maxChoice {
		fmt.Println("| ", mc)
	}
	fmt.Println("*-------------------")
}

func run() error {
	args := getArgs()
	if err := validateArgs(args); err != nil {
		return err
	}

	voteManager := NewVoteManager(args.votingTime, args.multiple, args.countdownTime, args.choice)
	handler, err := livechathandler.New(args.videoID, livechathandler.WithIntervalHandler(voteManager))
	if err != nil {
		return err
	}

	voteManager.PrintPreMessage()
	// polling end with press enter
	voteManager.StartInterruptionProcess()

	handler.Polling(voteManager.Ctx(), voteManager)

	voteManager.PrintEndMessage()
	voteManager.PrintVoteResultMessage()

	fmt.Println()

	return nil
}

func getArgs() (args arguments) {
	flag.StringVar(&args.videoID, "id", "", "youtube video id")
	flag.IntVar(&args.votingTime, "t", 30, "voting time. default 30sec")
	flag.IntVar(&args.countdownTime, "s", 0, "wait time before start voting. default 0sec")
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
