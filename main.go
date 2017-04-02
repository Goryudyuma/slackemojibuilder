package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/davecgh/go-spew/spew"
	"github.com/mattn/go-shellwords"
	"github.com/nlopes/slack"
)

func runCmdStr(cmdstr string) error {
	// 文字列をコマンド、オプション単位でスライス化する
	c, err := shellwords.Parse(cmdstr)
	if err != nil {
		return err
	}
	spew.Dump(c)

	switch len(c) {
	case 0:
		// 空の文字列が渡された場合
		return nil
	case 1:
		// コマンドのみを渡された場合
		err = exec.Command(c[0]).Run()
	default:
		// コマンド+オプションを渡された場合
		// オプションは可変長でexec.Commandに渡す
		err = exec.Command(c[0], c[1:]...).Run()
	}
	if err != nil {
		return err
	}
	return nil
}

func main() {
	api := slack.New(os.Getenv("SLACK_API_TOKEN"))
	logger := log.New(os.Stdout, "slack-bot: ", log.Lshortfile|log.LstdFlags)
	slack.SetLogger(logger)
	//api.SetDebug(true)

	rtm := api.NewRTM()
	go rtm.ManageConnection()

	escape := strings.NewReplacer("\"", "\\\"", "\\", "\\\\")
	filenameEscape := regexp.MustCompile(`[\w]{0,20}`)
	pwd, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println(pwd)
	err = os.Setenv("EMOJI_DIR", fmt.Sprintf("%s/pic/", pwd))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for msg := range rtm.IncomingEvents {
		fmt.Print("Event Received: ")
		switch ev := msg.Data.(type) {
		case *slack.MessageEvent:
			fmt.Printf("Message: %v\n", ev)
			str := strings.SplitN(ev.Text, " ", 2)
			if len(str) != 2 {
				continue
			}
			if !filenameEscape.Match([]byte(str[1])) {
				continue
			}
			if utf8.RuneCountInString(str[1]) > 4 {
				for i := 0; i < 30; i++ {
					err = runCmdStr(fmt.Sprintf("convert -font %s/font/NotoSansCJKjp-Medium.otf  -pointsize  30 -gravity West -annotate -%d+0 '%s%s' %s/init.png pic/%s_%05d.png", pwd, i*utf8.RuneCountInString(str[1]), str[1], str[1], pwd, escape.Replace(str[0]), i))
					if err != nil {
						spew.Dump(err)
						continue
					}
				}
				runCmdStr(fmt.Sprintf("convert -delay 10 %s/pic/%s_*.png -loop 0 -layers optimize %s/pic/%s.gif", pwd, escape.Replace(str[0]), pwd, escape.Replace(str[0])))

				stdoutStderr, err := exec.Command(fmt.Sprintf("%s/slack-emojinator/upload.py", pwd), fmt.Sprintf("%s/pic/%s.gif", pwd, escape.Replace(str[0]))).CombinedOutput()
				spew.Dump(string(stdoutStderr))
				if err != nil {
					spew.Dump(err)
					continue
				}
			} else {
				err = runCmdStr(fmt.Sprintf("convert -font %s/font/NotoSansCJKjp-Medium.otf -pointsize  30 label:'%s' %s/pic/%s.png", pwd, str[1], pwd, escape.Replace(str[0])))
				if err != nil {
					spew.Dump(err)
					continue
				}
				stdoutStderr, err := exec.Command(fmt.Sprintf("%s/slack-emojinator/upload.py", pwd), fmt.Sprintf("%s/pic/%s.png", pwd, escape.Replace(str[0]))).CombinedOutput()
				spew.Dump(string(stdoutStderr))
				if err != nil {
					spew.Dump(err)
					continue
				}
			}

			a, _ := rtm.GetEmoji()
			spew.Dump(a)

		case *slack.RTMError:
			//fmt.Printf("Error: %s\n", ev.Error())

		case *slack.InvalidAuthEvent:
			//fmt.Printf("Invalid credentials")
			return

		default:

			//fmt.Printf("Unexpected: %v\n", msg.Data)
		}
	}
}
